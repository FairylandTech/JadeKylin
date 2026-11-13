/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-11 23:30:00 America/Los_Angeles
 *****************************************************************************/

// Package repository 数据访问层。这里用内存实现演示，
// 替换为 MongoDB / MySQL 时只需实现同样的接口。
package repository

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"jkylin.com/kylin/v2/internal/model"
)

var (
	ErrNotFound  = errors.New("record not found")
	ErrDuplicate = errors.New("duplicate record")
)

type ExampleRepository interface {
	Create(ctx context.Context, e *model.Example) error
	Get(ctx context.Context, id int64) (*model.Example, error)
	List(ctx context.Context, keyword string, offset, limit int) ([]*model.Example, int64, error)
	Update(ctx context.Context, e *model.Example) error
	Delete(ctx context.Context, id int64) error
}

type memoryExampleRepository struct {
	mu     sync.RWMutex
	nextID int64
	data   map[int64]*model.Example
}

func NewMemoryExampleRepository() ExampleRepository {
	return &memoryExampleRepository{data: make(map[int64]*model.Example)}
}

func (r *memoryExampleRepository) Create(_ context.Context, e *model.Example) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.nameExists(e.Name, 0) {
		return ErrDuplicate
	}
	r.nextID++
	now := time.Now()
	e.ID, e.CreatedAt, e.UpdatedAt = r.nextID, now, now
	cp := *e
	r.data[e.ID] = &cp
	return nil
}

func (r *memoryExampleRepository) Get(_ context.Context, id int64) (*model.Example, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	e, ok := r.data[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *e
	return &cp, nil
}

func (r *memoryExampleRepository) List(_ context.Context, keyword string, offset, limit int) ([]*model.Example, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keyword = strings.ToLower(strings.TrimSpace(keyword))
	matched := make([]*model.Example, 0, len(r.data))
	for _, e := range r.data {
		if keyword == "" || strings.Contains(strings.ToLower(e.Name), keyword) {
			cp := *e
			matched = append(matched, &cp)
		}
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].ID > matched[j].ID })

	total := int64(len(matched))
	if offset >= len(matched) {
		return []*model.Example{}, total, nil
	}
	end := min(offset+limit, len(matched))
	return matched[offset:end], total, nil
}

func (r *memoryExampleRepository) Update(_ context.Context, e *model.Example) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.data[e.ID]; !ok {
		return ErrNotFound
	}
	if r.nameExists(e.Name, e.ID) {
		return ErrDuplicate
	}
	e.UpdatedAt = time.Now()
	cp := *e
	r.data[e.ID] = &cp
	return nil
}

func (r *memoryExampleRepository) Delete(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.data[id]; !ok {
		return ErrNotFound
	}
	delete(r.data, id)
	return nil
}

// nameExists 调用方需持有锁。
func (r *memoryExampleRepository) nameExists(name string, excludeID int64) bool {
	for id, e := range r.data {
		if id != excludeID && strings.EqualFold(e.Name, name) {
			return true
		}
	}
	return false
}
