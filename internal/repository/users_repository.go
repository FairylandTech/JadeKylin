/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 07:53:20 America/Los_Angeles
 *****************************************************************************/

package repository

import (
	"context"
	"maps"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"jkylin.com/kylin/v2/internal/model"
	"jkylin.com/kylin/v2/internal/pkg/support"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (userRepository *UserRepository) Create(ctx context.Context, record *model.User) error {
	return support.DBError(userRepository.db.WithContext(ctx).Create(record).Error)
}

func (userRepository *UserRepository) Get(ctx context.Context, id int64) (*model.User, error) {
	return support.FirstRecord[model.User](support.EnabledRecords(ctx, userRepository.db).
		Omit("password").
		Where("id = ?", id))
}

func (userRepository *UserRepository) List(ctx context.Context, keyword string, offset, limit int) ([]model.User, int64, error) {
	query := support.EnabledRecords(ctx, userRepository.db).Omit("password")
	if keyword != "" {
		// ILIKE 能命中 migration 中的 pg_trgm GIN 索引；strpos(lower(...)) 只能全表扫描。
		pattern := support.ContainsPattern(keyword)
		query = query.Where(`(username ILIKE ? ESCAPE '\' OR nickname ILIKE ? ESCAPE '\')`, pattern, pattern)
	}
	return support.ListRecords[model.User](query, offset, limit)
}

func (userRepository *UserRepository) Update(ctx context.Context, id int64, version *int, changes map[string]any) (*model.User, error) {
	var record model.User
	query := support.EnabledRecords(ctx, userRepository.db).Model(&record).Where("id = ?", id)
	if version != nil {
		query = query.Where("version = ?", *version)
	}
	updates := make(map[string]any, len(changes)+1)
	maps.Copy(updates, changes)
	updates["version"] = gorm.Expr("version + 1")
	result := query.Clauses(clause.Returning{Columns: userReturnColumns}).Updates(updates)
	if result.Error != nil {
		return nil, support.DBError(result.Error)
	}
	if result.RowsAffected > 0 {
		record.Password = ""
		return &record, nil
	}
	if version == nil {
		return nil, support.ErrNotFound
	}
	if err := support.EnabledRecords(ctx, userRepository.db).Select("id").First(&model.User{}, "id = ?", id).Error; err != nil {
		return nil, support.DBError(err)
	}
	return nil, support.ErrVersionConflict
}

// GetCredentials 按用户名读取启用用户（包含密码哈希），仅供登录校验使用。
func (userRepository *UserRepository) GetCredentials(ctx context.Context, username string) (*model.User, error) {
	return support.FirstRecord[model.User](support.EnabledRecords(ctx, userRepository.db).Where("username = ?", username))
}

// PermissionCodes 返回启用且状态正常的用户通过启用角色拥有的全部权限 code（去重）。
func (userRepository *UserRepository) PermissionCodes(ctx context.Context, userID int64) ([]string, error) {
	codes := make([]string, 0)
	err := userRepository.db.WithContext(ctx).Raw(`SELECT DISTINCT permission
FROM kylin_v2_users u
JOIN kylin_v2_roles r ON r.code = ANY (u.roles) AND r.enabled = 'Y'
CROSS JOIN LATERAL unnest(r.permissions) AS permission
WHERE u.id = ? AND u.enabled = 'Y' AND u.status = 'Y'`, userID).Scan(&codes).Error
	return codes, support.DBError(err)
}

// ExistsByUsername 判断是否存在启用的同名用户。
func (userRepository *UserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	err := support.EnabledRecords(ctx, userRepository.db).Model(&model.User{}).Where("username = ?", username).Limit(1).Count(&count).Error
	return count > 0, support.DBError(err)
}

// GetPasswordHash 返回启用用户的密码哈希，仅供密码校验使用。
func (userRepository *UserRepository) GetPasswordHash(ctx context.Context, id int64) (string, error) {
	var hash string
	result := support.EnabledRecords(ctx, userRepository.db).Model(&model.User{}).Where("id = ?", id).Limit(1).Pluck("password", &hash)
	if result.Error != nil {
		return "", support.DBError(result.Error)
	}
	if result.RowsAffected == 0 {
		return "", support.ErrNotFound
	}
	return hash, nil
}

// UpdatePassword 写入新密码哈希并递增版本；expectedHash 非空时要求当前哈希未被修改。
func (userRepository *UserRepository) UpdatePassword(ctx context.Context, id int64, expectedHash *string, hash string) error {
	query := support.EnabledRecords(ctx, userRepository.db).Model(&model.User{}).Where("id = ?", id)
	if expectedHash != nil {
		query = query.Where("password = ?", *expectedHash)
	}
	return support.WriteError(query.Updates(map[string]any{"password": hash, "version": gorm.Expr("version + 1")}))
}

func (userRepository *UserRepository) Delete(ctx context.Context, id int64) error {
	result := support.EnabledRecords(ctx, userRepository.db).
		Model(&model.User{}).
		Where("id = ?", id).
		Updates(map[string]any{"enabled": "N", "version": gorm.Expr("version + 1")})
	return support.WriteError(result)
}

var userReturnColumns = []clause.Column{
	{Name: "id"},
	{Name: "username"},
	{Name: "nickname"},
	{Name: "remark"},
	{Name: "status"},
	{Name: "roles"},
	{Name: "created_at"},
	{Name: "updated_at"},
	{Name: "enabled"},
	{Name: "version"},
}
