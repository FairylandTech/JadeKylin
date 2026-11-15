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

	"jkylin.com/kylin/v2/internal/module/auth"
	"jkylin.com/kylin/v2/internal/module/permission"
	"jkylin.com/kylin/v2/internal/module/role"
	"jkylin.com/kylin/v2/internal/module/user"
	"jkylin.com/kylin/v2/internal/pkg/authorization"
)

type Services struct {
	Auth        *auth.Service
	Users       *user.Service
	Roles       *role.Service
	Permissions *permission.Service
}

// NewServices 组装登录、用户、角色和权限服务，并接好会话下线与权限判断。要求 dependencies.Enabled()。
func NewServices(dependencies Dependencies) *Services {
	userRepository := user.NewRepository(dependencies.Database)
	roleRepository := role.NewRepository(dependencies.Database)
	permissionRepository := permission.NewRepository(dependencies.Database)
	authorizationCache := authorization.NewCache(dependencies.Redis, time.Duration(dependencies.Auth.IdleTimeout)*time.Second)
	authorizer := authorization.NewAuthorizer(authorizationCache, userRepository)

	authService := auth.NewService(userRepository, auth.NewStore(dependencies.Redis), dependencies.Auth)
	userService := user.NewService(userRepository, roleRepository, authorizer)
	userService.SessionRevoker = authService // 改密码 / 重置密码 / 禁用 / 删除用户时让该用户所有设备下线
	return &Services{
		Auth:        authService,
		Users:       userService,
		Roles:       role.NewService(roleRepository, permissionRepository, authorizer),
		Permissions: permission.NewService(permissionRepository, authorizer),
	}
}
