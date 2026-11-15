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

package user

import (
	"context"
	"maps"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"jkylin.com/kylin/v2/internal/pkg/persistence"
)

// returnColumns 是更新后返回的列：不包含 password，避免密码哈希离开数据库。
var returnColumns = []clause.Column{
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

type Repository struct {
	database *gorm.DB
}

func NewRepository(database *gorm.DB) *Repository {
	return &Repository{database: database}
}

func (userRepository *Repository) Create(ctx context.Context, record *User) error {
	return persistence.TranslateError(userRepository.database.WithContext(ctx).Create(record).Error)
}

func (userRepository *Repository) Get(ctx context.Context, id int64) (*User, error) {
	return persistence.FirstRecord[User](persistence.EnabledRecords(ctx, userRepository.database).
		Omit("password").
		Where("id = ?", id))
}

func (userRepository *Repository) List(ctx context.Context, keyword string, offset, limit int) ([]User, int64, error) {
	query := persistence.EnabledRecords(ctx, userRepository.database).Omit("password")
	if keyword != "" {
		// ILIKE 能命中 migration 中的 pg_trgm GIN 索引；strpos(lower(...)) 只能全表扫描。
		pattern := persistence.ContainsPattern(keyword)
		query = query.Where(`(username ILIKE ? ESCAPE '\' OR nickname ILIKE ? ESCAPE '\')`, pattern, pattern)
	}
	return persistence.ListRecords[User](query, offset, limit)
}

func (userRepository *Repository) Update(ctx context.Context, id int64, version *int, changes map[string]any) (*User, error) {
	var record User
	query := persistence.EnabledRecords(ctx, userRepository.database).Model(&record).Where("id = ?", id)
	if version != nil {
		query = query.Where("version = ?", *version)
	}
	updates := make(map[string]any, len(changes)+1)
	maps.Copy(updates, changes)
	updates["version"] = gorm.Expr("version + 1")
	result := query.Clauses(clause.Returning{Columns: returnColumns}).Updates(updates)
	if result.Error != nil {
		return nil, persistence.TranslateError(result.Error)
	}
	if result.RowsAffected > 0 {
		record.Password = ""
		return &record, nil
	}
	if version == nil {
		return nil, persistence.ErrNotFound
	}
	if err := persistence.EnabledRecords(ctx, userRepository.database).Select("id").First(&User{}, "id = ?", id).Error; err != nil {
		return nil, persistence.TranslateError(err)
	}
	return nil, persistence.ErrVersionConflict
}

// GetCredentials 按用户名读取启用用户（包含密码哈希），仅供登录校验使用。
func (userRepository *Repository) GetCredentials(ctx context.Context, username string) (*User, error) {
	return persistence.FirstRecord[User](persistence.EnabledRecords(ctx, userRepository.database).Where("username = ?", username))
}

// PermissionCodes 返回启用且状态正常的用户通过启用角色拥有的全部权限 code（去重）。
func (userRepository *Repository) PermissionCodes(ctx context.Context, userID int64) ([]string, error) {
	codes := make([]string, 0)
	err := userRepository.database.WithContext(ctx).Raw(`SELECT DISTINCT permission
FROM kylin_v2_users AS user_record
JOIN kylin_v2_roles AS role_record ON role_record.code = ANY (user_record.roles) AND role_record.enabled = 'Y'
CROSS JOIN LATERAL unnest(role_record.permissions) AS permission
WHERE user_record.id = ? AND user_record.enabled = 'Y' AND user_record.status = 'Y'`, userID).Scan(&codes).Error
	return codes, persistence.TranslateError(err)
}

// ExistsByUsername 判断是否存在启用的同名用户。
func (userRepository *Repository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	err := persistence.EnabledRecords(ctx, userRepository.database).Model(&User{}).Where("username = ?", username).Limit(1).Count(&count).Error
	return count > 0, persistence.TranslateError(err)
}

// GetPasswordHash 返回启用用户的密码哈希，仅供密码校验使用。
func (userRepository *Repository) GetPasswordHash(ctx context.Context, id int64) (string, error) {
	var hash string
	result := persistence.EnabledRecords(ctx, userRepository.database).Model(&User{}).Where("id = ?", id).Limit(1).Pluck("password", &hash)
	if result.Error != nil {
		return "", persistence.TranslateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return "", persistence.ErrNotFound
	}
	return hash, nil
}

// UpdatePassword 写入新密码哈希并递增版本；expectedHash 非空时要求当前哈希未被修改。
func (userRepository *Repository) UpdatePassword(ctx context.Context, id int64, expectedHash *string, hash string) error {
	query := persistence.EnabledRecords(ctx, userRepository.database).Model(&User{}).Where("id = ?", id)
	if expectedHash != nil {
		query = query.Where("password = ?", *expectedHash)
	}
	return persistence.WriteError(query.Updates(map[string]any{"password": hash, "version": gorm.Expr("version + 1")}))
}

func (userRepository *Repository) Delete(ctx context.Context, id int64) error {
	result := persistence.EnabledRecords(ctx, userRepository.database).
		Model(&User{}).
		Where("id = ?", id).
		Updates(map[string]any{"enabled": "N", "version": gorm.Expr("version + 1")})
	return persistence.WriteError(result)
}
