/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 23:30:00 America/Los_Angeles
 *****************************************************************************/

package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"jkylin.com/kylin/v2/internal/config"
	"jkylin.com/kylin/v2/internal/contracts/requests"
	"jkylin.com/kylin/v2/internal/model"
	"jkylin.com/kylin/v2/internal/pkg/support"
	"jkylin.com/kylin/v2/internal/repository"
	"jkylin.com/kylin/v2/pkg/captcha"
	"jkylin.com/kylin/v2/pkg/errcode"
	"jkylin.com/kylin/v2/pkg/logger"
)

var (
	errInvalidCredentials = errcode.Unauthorized.WithMessage("invalid username or password")
	errInvalidToken       = errcode.Unauthorized.WithMessage("token is invalid or expired")
	errSessionGone        = errcode.Unauthorized.WithMessage("session expired or logged out")
	errInvalidCaptcha     = errcode.InvalidParams.WithMessage("captcha is incorrect or expired")
)

// dummyPasswordHash 用于用户名不存在时仍执行一次 bcrypt 比较，避免通过响应耗时枚举用户名。
var dummyPasswordHash = sync.OnceValue(func() []byte {
	hash, _ := bcrypt.GenerateFromPassword([]byte("kylin-dummy-password"), bcrypt.DefaultCost)
	return hash
})

type accessClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type AuthService struct {
	UserRepository *repository.UserRepository
	AuthStore      *repository.AuthStore
	cfg            config.Auth
	secret         []byte
	now            func() time.Time
}

func NewAuthService(userRepository *repository.UserRepository, authStore *repository.AuthStore, cfg config.Auth) *AuthService {
	return &AuthService{UserRepository: userRepository, AuthStore: authStore, cfg: cfg, secret: []byte(cfg.JWTSecret), now: time.Now}
}

// SetClock 替换时间源（测试用）。
func (service *AuthService) SetClock(now func() time.Time) { service.now = now }

func (service *AuthService) tokenTTL() time.Duration {
	return time.Duration(service.cfg.TokenTTL) * time.Second
}

func (service *AuthService) idleTimeout() time.Duration {
	return time.Duration(service.cfg.IdleTimeout) * time.Second
}

// Captcha 生成 4 位数字验证码，保存到 Redis，并直接返回验证码明文。
func (service *AuthService) Captcha(ctx context.Context) (*requests.CaptchaResponse, error) {
	code, err := captcha.NewCode()
	if err != nil {
		return nil, errcode.Internal.Wrap(err)
	}
	id, err := randomID()
	if err != nil {
		return nil, errcode.Internal.Wrap(err)
	}
	if err := service.AuthStore.SaveCaptcha(ctx, id, code, time.Duration(service.cfg.CaptchaTTL)*time.Second); err != nil {
		return nil, errcode.Internal.Wrap(err)
	}
	return &requests.CaptchaResponse{CaptchaID: id, Code: code, ExpiresIn: service.cfg.CaptchaTTL}, nil
}

// Login 校验验证码和密码，签发 JWT 并在 Redis 中创建会话。
func (service *AuthService) Login(ctx context.Context, requestBody *requests.LoginRequest, clientIP, userAgent string) (*requests.LoginResponse, error) {
	// 验证码一次性：无论后续是否登录成功都已被消费。
	expected, err := service.AuthStore.TakeCaptcha(ctx, requestBody.CaptchaID)
	if err != nil {
		return nil, errcode.Internal.Wrap(err)
	}
	if expected == "" || subtle.ConstantTimeCompare([]byte(expected), []byte(requestBody.CaptchaCode)) != 1 {
		return nil, errInvalidCaptcha
	}

	user, err := service.UserRepository.GetCredentials(ctx, strings.TrimSpace(requestBody.Username))
	if errors.Is(err, support.ErrNotFound) {
		_ = bcrypt.CompareHashAndPassword(dummyPasswordHash(), []byte(requestBody.Password))
		return nil, errInvalidCredentials
	}
	if err != nil {
		return nil, errcode.Internal.Wrap(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(requestBody.Password)) != nil {
		return nil, errInvalidCredentials
	}
	user.Password = ""
	if user.Status != "Y" {
		return nil, errcode.Forbidden.WithMessage("user is disabled")
	}

	sessionID, err := randomID()
	if err != nil {
		return nil, errcode.Internal.Wrap(err)
	}
	now := service.now()
	token, err := service.signToken(user.ID, user.Username, sessionID, now)
	if err != nil {
		return nil, errcode.Internal.Wrap(err)
	}
	evicted, err := service.AuthStore.CreateSession(ctx, &repository.AuthSession{
		SessionID: sessionID, UserID: user.ID, Username: user.Username, Token: token,
		ClientIP: clientIP, UserAgent: userAgent, LoginAt: now,
	}, service.idleTimeout(), service.cfg.MaxSessions)
	if err != nil {
		return nil, errcode.Internal.Wrap(err)
	}
	if len(evicted) > 0 {
		logger.FromContext(ctx).Info("login evicted previous sessions", zap.Int64("user_id", user.ID), zap.Strings("session_ids", evicted))
	}
	return &requests.LoginResponse{AccessToken: token, TokenType: "Bearer", ExpiresIn: service.cfg.TokenTTL, User: user}, nil
}

// Authenticate 校验 JWT 和 Redis 会话，并为会话续期。
// JWT 剩余有效期不足 idle_timeout 时签发新 JWT（同一会话），通过 renewedToken 返回；
// 这样只要会话还活跃，客户端手中的 JWT 就不会先于会话过期。
func (service *AuthService) Authenticate(ctx context.Context, rawToken string) (*model.Principal, string, error) {
	now := service.now()
	var claims accessClaims
	_, err := jwt.ParseWithClaims(rawToken, &claims, func(*jwt.Token) (any, error) { return service.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(service.cfg.Issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(func() time.Time { return now }),
	)
	if err != nil || claims.ID == "" {
		return nil, "", errInvalidToken
	}
	// user_id 与 sub 必须一致，防止只篡改其中一个字段（签名已保证不可篡改，这里做防御性校验）。
	userID := claims.UserID
	if userID <= 0 || claims.Subject != strconv.FormatInt(userID, 10) {
		return nil, "", errInvalidToken
	}

	renewed := ""
	if claims.ExpiresAt.Sub(now) < service.idleTimeout() {
		if renewed, err = service.signToken(userID, claims.Username, claims.ID, now); err != nil {
			return nil, "", errcode.Internal.Wrap(err)
		}
	}
	alive, err := service.AuthStore.TouchSession(ctx, userID, claims.ID, service.idleTimeout(), renewed)
	if err != nil {
		return nil, "", errcode.Internal.Wrap(err)
	}
	if !alive {
		return nil, "", errSessionGone
	}
	return &model.Principal{UserID: userID, Username: claims.Username, SessionID: claims.ID}, renewed, nil
}

// LocalSessionID 是本地 / 开发模式跳过认证时使用的会话 id（Redis 中不存在对应会话）。
const LocalSessionID = "local"

// LocalPrincipal 按用户名查找启用的用户，作为本地 / 开发模式跳过认证时的默认身份。
func (service *AuthService) LocalPrincipal(ctx context.Context, username string) (*model.Principal, error) {
	user, err := service.UserRepository.GetCredentials(ctx, username)
	switch {
	case errors.Is(err, support.ErrNotFound):
		return nil, errcode.Unauthorized.WithMessage("local auth user %q not found; set KYLIN_ADMIN_USERNAME / KYLIN_ADMIN_PASSWORD to create it", username)
	case err != nil:
		return nil, errcode.Internal.Wrap(err)
	}
	return &model.Principal{UserID: user.ID, Username: user.Username, SessionID: LocalSessionID}, nil
}

// Logout 删除当前会话，JWT 随之失效。
func (service *AuthService) Logout(ctx context.Context, principal *model.Principal) error {
	if err := service.AuthStore.DeleteSession(ctx, principal.UserID, principal.SessionID); err != nil {
		return errcode.Internal.Wrap(err)
	}
	return nil
}

// Me 返回当前登录用户。
func (service *AuthService) Me(ctx context.Context, principal *model.Principal) (*model.User, error) {
	record, err := service.UserRepository.Get(ctx, principal.UserID)
	return record, translateUserError(err)
}

// RevokeUser 让用户所有设备下线。
func (service *AuthService) RevokeUser(ctx context.Context, userID int64) error {
	return service.AuthStore.DeleteUserSessions(ctx, userID)
}

func (service *AuthService) signToken(userID int64, username, sessionID string, now time.Time) (string, error) {
	claims := accessClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    service.cfg.Issuer,
			Subject:   strconv.FormatInt(userID, 10),
			ID:        sessionID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(service.tokenTTL())),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(service.secret)
}

func randomID() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}
