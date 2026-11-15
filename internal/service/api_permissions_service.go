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

package service

import (
	"context"
	"errors"
	"fmt"

	"jkylin.com/kylin/v2/internal/contracts/requests"
	"jkylin.com/kylin/v2/internal/model"
	"jkylin.com/kylin/v2/internal/model/valueobject"
	"jkylin.com/kylin/v2/internal/pkg/support"
	"jkylin.com/kylin/v2/internal/repository"
	"jkylin.com/kylin/v2/pkg/errcode"
)

type ApiPermissionService struct {
	ApiPermissionRepository *repository.ApiPermissionRepository
	PermissionRepository    *repository.PermissionRepository
	UserRepository          *repository.UserRepository
	AuthzCache              *repository.AuthzCache
}

func NewApiPermissionService(apiPermissionRepository *repository.ApiPermissionRepository, permissionRepository *repository.PermissionRepository,
	userRepository *repository.UserRepository, authzCache *repository.AuthzCache) *ApiPermissionService {
	return &ApiPermissionService{
		ApiPermissionRepository: apiPermissionRepository,
		PermissionRepository:    permissionRepository,
		UserRepository:          userRepository,
		AuthzCache:              authzCache,
	}
}

func (service *ApiPermissionService) List(ctx context.Context) ([]model.ApiPermission, error) {
	records, err := service.ApiPermissionRepository.List(ctx)
	if err != nil {
		return nil, errcode.Internal.Wrap(err)
	}
	return records, nil
}

// Update 修改接口的放行权限；permission 必须是已启用的权限 code。
func (service *ApiPermissionService) Update(ctx context.Context, id int64, requestBody *requests.UpdateApiPermissionRequest) (*model.ApiPermission, error) {
	changes := make(map[string]any)
	if requestBody.Permission != nil {
		permissions, err := normalizeCodes(ctx, "permission", valueobject.StringArray{*requestBody.Permission}, service.PermissionRepository.ExistingCodes)
		if err != nil {
			return nil, err
		}
		changes["permission"] = permissions[0]
	}
	if requestBody.Description != nil {
		changes["description"] = *requestBody.Description
	}
	if len(changes) == 0 {
		return nil, errcode.InvalidParams.WithMessage("at least one api permission field is required")
	}
	record, err := service.ApiPermissionRepository.Update(ctx, id, changes)
	switch {
	case errors.Is(err, support.ErrNotFound):
		return nil, errcode.NotFound.WithMessage("api permission not found")
	case err != nil:
		return nil, errcode.Internal.Wrap(err)
	}
	if err := service.SyncCache(ctx); err != nil {
		return nil, errcode.Internal.WithMessage("api permission updated but failed to sync cache").Wrap(err)
	}
	return record, nil
}

// SyncCache 把数据库中最新的接口放行权限同步到 Redis（启动时、表数据变更后调用）。
func (service *ApiPermissionService) SyncCache(ctx context.Context) error {
	records, err := service.ApiPermissionRepository.List(ctx)
	if err != nil {
		return err
	}
	syncID, err := randomID()
	if err != nil {
		return err
	}
	return service.AuthzCache.SyncAPIPermissions(ctx, records, syncID)
}

// Authorize 判断用户能否访问接口（供 auth.Authorization 中间件使用）。
// 数据全部从 Redis 读取；缓存缺失时（首次请求、Redis 重启、缓存被失效）才从数据库加载一次。
func (service *ApiPermissionService) Authorize(ctx context.Context, userID int64, method, path string) (mapped, allowed bool, err error) {
	for range 3 { // 最多补两次缓存：接口权限表、用户权限
		result, err := service.AuthzCache.Check(ctx, userID, method, path)
		if err != nil {
			return false, false, err
		}
		switch result {
		case repository.AuthzAPIMissing:
			if err := service.SyncCache(ctx); err != nil {
				return false, false, err
			}
		case repository.AuthzUserMissing:
			permissions, err := service.UserRepository.PermissionCodes(ctx, userID)
			if err != nil {
				return false, false, err
			}
			if err := service.AuthzCache.SetUserPermissions(ctx, userID, permissions); err != nil {
				return false, false, err
			}
		case repository.AuthzNotMapped:
			return false, false, nil
		case repository.AuthzDenied:
			return true, false, nil
		case repository.AuthzAllowed:
			return true, true, nil
		default:
			return false, false, fmt.Errorf("unexpected authz cache result %d", result)
		}
	}
	return false, false, errors.New("authz cache did not settle")
}
