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
	"strings"
	"sync"

	"jkylin.com/kylin/v2/internal/middleware/security"
	"jkylin.com/kylin/v2/internal/module/auth"
	"jkylin.com/kylin/v2/internal/pkg/identity"
)

// localPrincipalResolver 查找跳过认证时使用的用户，成功后缓存（失败时下次请求重试）。
func localPrincipalResolver(authService *auth.Service, username string) security.PrincipalResolver {
	username = strings.TrimSpace(username)
	if username == "" {
		username = DefaultLocalUsername
	}
	var mutex sync.Mutex
	var cached *identity.Principal
	return func(ctx context.Context) (*identity.Principal, error) {
		mutex.Lock()
		defer mutex.Unlock()
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
