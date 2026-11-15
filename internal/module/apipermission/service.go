/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-15 01:30:00 America/Los_Angeles
 *****************************************************************************/

package apipermission

import (
	"context"
	"errors"
	"fmt"

	"jkylin.com/kylin/v2/internal/module/permission"
	"jkylin.com/kylin/v2/internal/module/user"
	"jkylin.com/kylin/v2/internal/pkg/identifier"
	"jkylin.com/kylin/v2/internal/pkg/persistence"
	"jkylin.com/kylin/v2/internal/pkg/validation"
	"jkylin.com/kylin/v2/internal/pkg/valueobject"
	"jkylin.com/kylin/v2/pkg/errcode"
)

type Service struct {
	ApiPermissionRepository *Repository
	PermissionRepository    *permission.Repository
	UserRepository          *user.Repository
	AuthorizationCache      *AuthorizationCache
}

func NewService(apiPermissionRepository *Repository, permissionRepository *permission.Repository,
	userRepository *user.Repository, authorizationCache *AuthorizationCache) *Service {
	return &Service{
		ApiPermissionRepository: apiPermissionRepository,
		PermissionRepository:    permissionRepository,
		UserRepository:          userRepository,
		AuthorizationCache:      authorizationCache,
	}
}

func (apiPermissionService *Service) List(ctx context.Context) ([]ApiPermission, error) {
	records, err := apiPermissionService.ApiPermissionRepository.List(ctx)
	if err != nil {
		return nil, errcode.Internal.Wrap(err)
	}
	return records, nil
}

// Update 修改接口的放行权限；permission 必须是已启用的权限 code。
func (apiPermissionService *Service) Update(ctx context.Context, id int64, requestBody *UpdateRequest) (*ApiPermission, error) {
	changes := make(map[string]any)
	if requestBody.Permission != nil {
		permissions, err := validation.NormalizeCodes(ctx, "permission", valueobject.StringArray{*requestBody.Permission},
			apiPermissionService.PermissionRepository.ExistingCodes)
		if err != nil {
			return nil, err
		}
		changes["permission"] = permissions[0]
	}
	if requestBody.Description != nil {
		changes["description"] = *requestBody.Description
	}
	if len(changes) == 0 {
		return nil, errcode.InvalidParameters.WithMessage("at least one api permission field is required")
	}
	record, err := apiPermissionService.ApiPermissionRepository.Update(ctx, id, changes)
	switch {
	case errors.Is(err, persistence.ErrNotFound):
		return nil, errcode.NotFound.WithMessage("api permission not found")
	case err != nil:
		return nil, errcode.Internal.Wrap(err)
	}
	if err := apiPermissionService.SyncCache(ctx); err != nil {
		return nil, errcode.Internal.WithMessage("api permission updated but failed to sync cache").Wrap(err)
	}
	return record, nil
}

// SyncCache 把数据库中最新的接口放行权限同步到 Redis（启动时、表数据变更后调用）。
func (apiPermissionService *Service) SyncCache(ctx context.Context) error {
	records, err := apiPermissionService.ApiPermissionRepository.List(ctx)
	if err != nil {
		return err
	}
	syncID, err := identifier.NewRandom()
	if err != nil {
		return err
	}
	return apiPermissionService.AuthorizationCache.SyncAPIPermissions(ctx, records, syncID)
}

// Authorize 判断用户能否访问接口（供 security.Authorization 中间件使用）。
// 数据全部从 Redis 读取；缓存缺失时（首次请求、Redis 重启、缓存被失效）才从数据库加载一次。
func (apiPermissionService *Service) Authorize(ctx context.Context, userID int64, method, path string) (mapped, allowed bool, err error) {
	for range authorizationCheckMaximumRounds {
		result, err := apiPermissionService.AuthorizationCache.Check(ctx, userID, method, path)
		if err != nil {
			return false, false, err
		}
		switch result {
		case AuthorizationAPIMissing:
			if err := apiPermissionService.SyncCache(ctx); err != nil {
				return false, false, err
			}
		case AuthorizationUserMissing:
			permissions, err := apiPermissionService.UserRepository.PermissionCodes(ctx, userID)
			if err != nil {
				return false, false, err
			}
			if err := apiPermissionService.AuthorizationCache.SetUserPermissions(ctx, userID, permissions); err != nil {
				return false, false, err
			}
		case AuthorizationNotMapped:
			return false, false, nil
		case AuthorizationDenied:
			return true, false, nil
		case AuthorizationAllowed:
			return true, true, nil
		default:
			return false, false, fmt.Errorf("unexpected authz cache result %d", result)
		}
	}
	return false, false, errors.New("authz cache did not settle")
}
