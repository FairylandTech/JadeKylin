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
	"time"

	"jkylin.com/kylin/v2/internal/module/apipermission"
	"jkylin.com/kylin/v2/internal/module/auth"
	"jkylin.com/kylin/v2/internal/module/permission"
	"jkylin.com/kylin/v2/internal/module/role"
	"jkylin.com/kylin/v2/internal/module/user"
)

type Services struct {
	Auth           *auth.Service
	Users          *user.Service
	Roles          *role.Service
	ApiPermissions *apipermission.Service
}

// NewServices 组装登录、用户、角色和接口权限服务，并接好会话下线与鉴权缓存失效。要求 dependencies.Enabled()。
func NewServices(dependencies Dependencies) *Services {
	userRepository := user.NewRepository(dependencies.Database)
	permissionRepository := permission.NewRepository(dependencies.Database)
	authorizationCache := apipermission.NewAuthorizationCache(dependencies.Redis, time.Duration(dependencies.Auth.IdleTimeout)*time.Second)

	authService := auth.NewService(userRepository, auth.NewStore(dependencies.Redis), dependencies.Auth)
	userService := user.NewService(userRepository, role.NewRepository(dependencies.Database))
	userService.SessionRevoker = authService // 改密码 / 重置密码 / 禁用 / 删除用户时让该用户所有设备下线
	userService.AuthorizationCache = authorizationCache
	roleService := role.NewService(role.NewRepository(dependencies.Database), permissionRepository)
	roleService.AuthorizationCache = authorizationCache
	apiPermissionService := apipermission.NewService(apipermission.NewRepository(dependencies.Database), permissionRepository, userRepository, authorizationCache)
	return &Services{
		Auth:           authService,
		Users:          userService,
		Roles:          roleService,
		ApiPermissions: apiPermissionService,
	}
}
