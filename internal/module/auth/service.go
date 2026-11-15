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

package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"jkylin.com/kylin/v2/internal/config"
	"jkylin.com/kylin/v2/internal/module/user"
	"jkylin.com/kylin/v2/internal/pkg/credential"
	"jkylin.com/kylin/v2/internal/pkg/identifier"
	"jkylin.com/kylin/v2/internal/pkg/identity"
	"jkylin.com/kylin/v2/internal/pkg/persistence"
	"jkylin.com/kylin/v2/pkg/captcha"
	"jkylin.com/kylin/v2/pkg/errcode"
	"jkylin.com/kylin/v2/pkg/logger"
)

const (
	// LocalSessionID 是本地 / 开发模式跳过认证时使用的会话 id（Redis 中不存在对应会话）。
	LocalSessionID = "local"
	// bearerTokenType 是登录响应中的 token 类型。
	bearerTokenType = "Bearer"
)

var (
	errInvalidCredentials = errcode.Unauthorized.WithMessage("invalid username or password")
	errInvalidToken       = errcode.Unauthorized.WithMessage("token is invalid or expired")
	errSessionGone        = errcode.Unauthorized.WithMessage("session expired or logged out")
	errInvalidCaptcha     = errcode.InvalidParameters.WithMessage("captcha is incorrect or expired")
)

type accessClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type Service struct {
	UserRepository *user.Repository
	SessionStore   *Store
	configuration  config.Auth
	secret         []byte
	now            func() time.Time
}

func NewService(userRepository *user.Repository, sessionStore *Store, configuration config.Auth) *Service {
	return &Service{
		UserRepository: userRepository,
		SessionStore:   sessionStore,
		configuration:  configuration,
		secret:         []byte(configuration.JWTSecret),
		now:            time.Now,
	}
}

// SetClock 替换时间源（测试用）。
func (authService *Service) SetClock(now func() time.Time) {
	authService.now = now
}

func (authService *Service) tokenTimeToLive() time.Duration {
	return time.Duration(authService.configuration.TokenTTL) * time.Second
}

func (authService *Service) idleTimeout() time.Duration {
	return time.Duration(authService.configuration.IdleTimeout) * time.Second
}

func (authService *Service) captchaTimeToLive() time.Duration {
	return time.Duration(authService.configuration.CaptchaTTL) * time.Second
}

// Captcha 生成 4 位数字验证码，保存到 Redis，并直接返回验证码明文。
func (authService *Service) Captcha(ctx context.Context) (*CaptchaResponse, error) {
	code, err := captcha.NewCode()
	if err != nil {
		return nil, errcode.Internal.Wrap(err)
	}
	captchaID, err := identifier.NewRandom()
	if err != nil {
		return nil, errcode.Internal.Wrap(err)
	}
	if err := authService.SessionStore.SaveCaptcha(ctx, captchaID, code, authService.captchaTimeToLive()); err != nil {
		return nil, errcode.Internal.Wrap(err)
	}
	return &CaptchaResponse{CaptchaID: captchaID, Code: code, ExpiresIn: authService.configuration.CaptchaTTL}, nil
}

// Login 校验验证码和密码，签发 JWT 并在 Redis 中创建会话。
func (authService *Service) Login(ctx context.Context, requestBody *LoginRequest, clientIP, userAgent string) (*LoginResponse, error) {
	// 验证码一次性：无论后续是否登录成功都已被消费。
	expected, err := authService.SessionStore.TakeCaptcha(ctx, requestBody.CaptchaID)
	if err != nil {
		return nil, errcode.Internal.Wrap(err)
	}
	if expected == "" || subtle.ConstantTimeCompare([]byte(expected), []byte(requestBody.CaptchaCode)) != 1 {
		return nil, errInvalidCaptcha
	}

	account, err := authService.UserRepository.GetCredentials(ctx, strings.TrimSpace(requestBody.Username))
	if errors.Is(err, persistence.ErrNotFound) {
		credential.CompareWithDummyPassword(requestBody.Password)
		return nil, errInvalidCredentials
	}
	if err != nil {
		return nil, errcode.Internal.Wrap(err)
	}
	if !credential.MatchesPassword(account.Password, requestBody.Password) {
		return nil, errInvalidCredentials
	}
	account.Password = ""
	if account.Status != "Y" {
		return nil, errcode.Forbidden.WithMessage("user is disabled")
	}

	sessionID, err := identifier.NewRandom()
	if err != nil {
		return nil, errcode.Internal.Wrap(err)
	}
	now := authService.now()
	token, err := authService.signToken(account.ID, account.Username, sessionID, now)
	if err != nil {
		return nil, errcode.Internal.Wrap(err)
	}
	evicted, err := authService.SessionStore.CreateSession(ctx, &Session{
		SessionID: sessionID,
		UserID:    account.ID,
		Username:  account.Username,
		Token:     token,
		ClientIP:  clientIP,
		UserAgent: userAgent,
		LoginAt:   now,
	}, authService.idleTimeout(), authService.configuration.MaxSessions)
	if err != nil {
		return nil, errcode.Internal.Wrap(err)
	}
	if len(evicted) > 0 {
		logger.FromContext(ctx).Info("login evicted previous sessions", zap.Int64("user_id", account.ID), zap.Strings("session_ids", evicted))
	}
	return &LoginResponse{
		AccessToken: token,
		TokenType:   bearerTokenType,
		ExpiresIn:   authService.configuration.TokenTTL,
		User:        account,
	}, nil
}

// Authenticate 校验 JWT 和 Redis 会话，并为会话续期。
// JWT 剩余有效期不足 idle_timeout 时签发新 JWT（同一会话），通过 renewedToken 返回；
// 这样只要会话还活跃，客户端手中的 JWT 就不会先于会话过期。
func (authService *Service) Authenticate(ctx context.Context, rawToken string) (*identity.Principal, string, error) {
	now := authService.now()
	var claims accessClaims
	_, err := jwt.ParseWithClaims(rawToken, &claims, authService.signingKey,
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(authService.configuration.Issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(func() time.Time {
			return now
		}),
	)
	if err != nil || claims.ID == "" {
		return nil, "", errInvalidToken
	}
	// user_id 与 sub 必须一致，防止只篡改其中一个字段（签名已保证不可篡改，这里做防御性校验）。
	userID := claims.UserID
	if userID <= 0 || claims.Subject != strconv.FormatInt(userID, 10) {
		return nil, "", errInvalidToken
	}

	renewedToken := ""
	if claims.ExpiresAt.Sub(now) < authService.idleTimeout() {
		if renewedToken, err = authService.signToken(userID, claims.Username, claims.ID, now); err != nil {
			return nil, "", errcode.Internal.Wrap(err)
		}
	}
	alive, err := authService.SessionStore.TouchSession(ctx, userID, claims.ID, authService.idleTimeout(), renewedToken)
	if err != nil {
		return nil, "", errcode.Internal.Wrap(err)
	}
	if !alive {
		return nil, "", errSessionGone
	}
	return &identity.Principal{UserID: userID, Username: claims.Username, SessionID: claims.ID}, renewedToken, nil
}

// LocalPrincipal 按用户名查找启用的用户，作为本地 / 开发模式跳过认证时的默认身份。
func (authService *Service) LocalPrincipal(ctx context.Context, username string) (*identity.Principal, error) {
	account, err := authService.UserRepository.GetCredentials(ctx, username)
	switch {
	case errors.Is(err, persistence.ErrNotFound):
		return nil, errcode.Unauthorized.WithMessage("local auth user %q not found; set KYLIN_ADMIN_USERNAME / KYLIN_ADMIN_PASSWORD to create it", username)
	case err != nil:
		return nil, errcode.Internal.Wrap(err)
	}
	return &identity.Principal{UserID: account.ID, Username: account.Username, SessionID: LocalSessionID}, nil
}

// Logout 删除当前会话，JWT 随之失效。
func (authService *Service) Logout(ctx context.Context, principal *identity.Principal) error {
	if err := authService.SessionStore.DeleteSession(ctx, principal.UserID, principal.SessionID); err != nil {
		return errcode.Internal.Wrap(err)
	}
	return nil
}

// Me 返回当前登录用户。
func (authService *Service) Me(ctx context.Context, principal *identity.Principal) (*user.User, error) {
	record, err := authService.UserRepository.Get(ctx, principal.UserID)
	return record, user.TranslateError(err)
}

// RevokeUser 让用户所有设备下线。
func (authService *Service) RevokeUser(ctx context.Context, userID int64) error {
	return authService.SessionStore.DeleteUserSessions(ctx, userID)
}

// signingKey 返回校验 JWT 签名使用的密钥。
func (authService *Service) signingKey(*jwt.Token) (any, error) {
	return authService.secret, nil
}

func (authService *Service) signToken(userID int64, username, sessionID string, now time.Time) (string, error) {
	claims := accessClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    authService.configuration.Issuer,
			Subject:   strconv.FormatInt(userID, 10),
			ID:        sessionID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(authService.tokenTimeToLive())),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(authService.secret)
}
