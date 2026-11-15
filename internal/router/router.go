/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-11 04:20:18 America/Los_Angeles
 *****************************************************************************/

package router

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
	"jkylin.com/kylin/v2/internal/config"
	authmw "jkylin.com/kylin/v2/internal/middleware/auth"
	"jkylin.com/kylin/v2/internal/model"
	"jkylin.com/kylin/v2/internal/repository"
	"jkylin.com/kylin/v2/internal/router/apipermissions"
	"jkylin.com/kylin/v2/internal/router/auth"
	"jkylin.com/kylin/v2/internal/router/permissions"
	"jkylin.com/kylin/v2/internal/router/roles"
	"jkylin.com/kylin/v2/internal/router/users"
	"jkylin.com/kylin/v2/internal/service"
	"jkylin.com/kylin/v2/pkg/response"
)

type Dependencies struct {
	DB    *gorm.DB
	Redis *redis.Client
	Auth  config.Auth // DB、Redis、JWTSecret 都具备时才注册 /auth 和管理接口
	// SkipAuth 为 true（RUN_MODE=local/dev）时，未携带 Authorization 的请求以 LocalUsername 用户身份访问。
	SkipAuth      bool
	LocalUsername string
}

// DefaultLocalUsername 是未设置 KYLIN_ADMIN_USERNAME 时，跳过认证所使用的用户名。
const DefaultLocalUsername = "admin"

// Enabled 返回登录及管理接口是否可用。
func (deps Dependencies) Enabled() bool {
	return deps.DB != nil && deps.Redis != nil && deps.Auth.JWTSecret != ""
}

func Init(engine *gin.Engine, deps Dependencies) {
	// 健康检查（已在访问日志中跳过）
	engine.GET("/healthz", func(ctx *gin.Context) { ctx.String(http.StatusOK, "ok") })

	engine.GET("/example", func(ctx *gin.Context) {
		response.Success(ctx, gin.H{"example": "Hello Gin Server."})
	})

	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	if !deps.Enabled() {
		return
	}
	services := NewServices(deps)
	authService, userService := services.Auth, services.Users

	// 认证：解析 JWT 得到 user id 写入上下文；鉴权：按接口放行权限判断当前用户能否访问（数据从 Redis 读取）。
	authentication := authmw.Authentication(authService)
	if deps.SkipAuth {
		authentication = authmw.LocalAuthentication(authService, localPrincipalResolver(authService, deps.LocalUsername))
	}
	authenticated := engine.Group("", authentication)
	auth.Router(engine.Group(""), authenticated, authService, userService)

	protected := authenticated.Group("", authmw.Authorization(services.ApiPermissions))
	users.Router(protected, userService)
	roles.Router(protected, services.Roles)
	permissions.Router(protected, deps.DB)
	apipermissions.Router(protected, services.ApiPermissions)
}

// localPrincipalResolver 查找跳过认证时使用的用户，成功后缓存（失败时下次请求重试）。
func localPrincipalResolver(authService *service.AuthService, username string) authmw.PrincipalResolver {
	username = strings.TrimSpace(username)
	if username == "" {
		username = DefaultLocalUsername
	}
	var mu sync.Mutex
	var cached *model.Principal
	return func(ctx context.Context) (*model.Principal, error) {
		mu.Lock()
		defer mu.Unlock()
		if cached != nil {
			return cached, nil
		}
		principal, err := authService.LocalPrincipal(ctx, username)
		if err != nil {
			return nil, err
		}
		cached = principal
		return cached, nil
	}
}

type Services struct {
	Auth           *service.AuthService
	Users          *service.UserService
	Roles          *service.RoleService
	ApiPermissions *service.ApiPermissionService
}

// NewServices 组装登录、用户、角色和接口权限服务，并接好会话下线与鉴权缓存失效。要求 deps.Enabled()。
func NewServices(deps Dependencies) *Services {
	userRepository := repository.NewUserRepository(deps.DB)
	permissionRepository := repository.NewPermissionRepository(deps.DB)
	authzCache := repository.NewAuthzCache(deps.Redis, time.Duration(deps.Auth.IdleTimeout)*time.Second)

	authService := service.NewAuthService(userRepository, repository.NewAuthStore(deps.Redis), deps.Auth)
	userService := service.NewUserService(userRepository, repository.NewRoleRepository(deps.DB))
	userService.SessionRevoker = authService // 改密码 / 重置密码 / 禁用 / 删除用户时让该用户所有设备下线
	userService.AuthzCache = authzCache
	roleService := service.NewRoleService(repository.NewRoleRepository(deps.DB), permissionRepository)
	roleService.AuthzCache = authzCache
	apiPermissionService := service.NewApiPermissionService(repository.NewApiPermissionRepository(deps.DB), permissionRepository, userRepository, authzCache)
	return &Services{Auth: authService, Users: userService, Roles: roleService, ApiPermissions: apiPermissionService}
}
