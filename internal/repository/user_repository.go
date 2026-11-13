/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"jkylin.com/kylin/v2/internal/model"
)

var (
	ErrNotFound        = errors.New("record not found")
	ErrDuplicate       = errors.New("duplicate record")
	ErrVersionConflict = errors.New("record version conflict")
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, record *model.User) error {
	return userDBError(r.db.WithContext(ctx).Create(record).Error)
}

func (r *UserRepository) Get(ctx context.Context, id int64) (*model.User, error) {
	var record model.User
	if err := r.db.WithContext(ctx).Omit("password").First(&record, id).Error; err != nil {
		return nil, userDBError(err)
	}
	return &record, nil
}

func (r *UserRepository) List(ctx context.Context, keyword string, offset, limit int) ([]model.User, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.User{})
	if keyword != "" {
		query = query.Where("strpos(lower(username), lower(?)) > 0 OR strpos(lower(nickname), lower(?)) > 0", keyword, keyword)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	records := make([]model.User, 0)
	err := query.Omit("password").Order("id DESC").Offset(offset).Limit(limit).Find(&records).Error
	return records, total, err
}

// Update 通过单条 UPDATE 修改指定列并返回该次写入的记录。
func (r *UserRepository) Update(ctx context.Context, id int64, version *int, changes map[string]any) (*model.User, error) {
	var record model.User
	query := r.db.WithContext(ctx).Model(&record).Where("id = ?", id)
	if version != nil {
		query = query.Where("version = ?", *version)
	}
	changes["version"] = gorm.Expr("version + 1")
	result := query.Clauses(clause.Returning{Columns: userReturnColumns}).Updates(changes)
	if result.Error != nil {
		return nil, userDBError(result.Error)
	}
	if result.RowsAffected > 0 {
		record.Password = ""
		return &record, nil
	}
	if version == nil {
		return nil, ErrNotFound
	}
	if _, err := r.Get(ctx, id); err != nil {
		return nil, err
	}
	return nil, ErrVersionConflict
}

func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Delete(&model.User{}, id)
	if result.Error != nil {
		return userDBError(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

var userReturnColumns = []clause.Column{
	{Name: "id"}, {Name: "username"}, {Name: "nickname"}, {Name: "remark"},
	{Name: "status"}, {Name: "roles"}, {Name: "created_at"}, {Name: "updated_at"},
	{Name: "enabled"}, {Name: "version"},
}

func userDBError(err error) error {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return ErrNotFound
	case errors.Is(err, gorm.ErrDuplicatedKey), errors.As(err, &pgErr) && pgErr.Code == "23505":
		return ErrDuplicate
	default:
		return err
	}
}
