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

// Package service 业务逻辑层：编排 repository，把底层错误翻译为 errcode。
package service

import (
	"context"
	"errors"
	"strings"

	"go.uber.org/zap"
	"jkylin.com/kylin/v2/internal/model"
	"jkylin.com/kylin/v2/internal/repository"
	"jkylin.com/kylin/v2/pkg/errcode"
	"jkylin.com/kylin/v2/pkg/logger"
)

type ExampleService struct {
	repo repository.ExampleRepository
}

func NewExampleService(repo repository.ExampleRepository) *ExampleService {
	return &ExampleService{repo: repo}
}

func (s *ExampleService) Create(ctx context.Context, req *model.CreateExampleRequest) (*model.Example, error) {
	e := &model.Example{
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
	}
	if e.Name == "" {
		return nil, errcode.InvalidParams.WithMessage("name must not be blank")
	}
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, translate(err, "name %q already exists", e.Name)
	}

	logger.FromContext(ctx).Info("example created", zap.Int64("id", e.ID), zap.String("name", e.Name))
	return e, nil
}

func (s *ExampleService) Get(ctx context.Context, id int64) (*model.Example, error) {
	e, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, translate(err, "")
	}
	return e, nil
}

func (s *ExampleService) List(ctx context.Context, q *model.ListExampleQuery) ([]*model.Example, int64, error) {
	q.Normalize()
	list, total, err := s.repo.List(ctx, q.Keyword, q.Offset(), q.Size)
	if err != nil {
		return nil, 0, translate(err, "")
	}
	logger.FromContext(ctx).Debug("example listed",
		zap.String("keyword", q.Keyword), zap.Int("page", q.Page), zap.Int64("total", total))
	return list, total, nil
}

func (s *ExampleService) Update(ctx context.Context, id int64, req *model.UpdateExampleRequest) (*model.Example, error) {
	e, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, translate(err, "")
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, errcode.InvalidParams.WithMessage("name must not be blank")
		}
		e.Name = name
	}
	if req.Description != nil {
		e.Description = strings.TrimSpace(*req.Description)
	}
	if err := s.repo.Update(ctx, e); err != nil {
		return nil, translate(err, "name %q already exists", e.Name)
	}

	logger.FromContext(ctx).Info("example updated", zap.Int64("id", e.ID))
	return e, nil
}

func (s *ExampleService) Delete(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return translate(err, "")
	}
	logger.FromContext(ctx).Info("example deleted", zap.Int64("id", id))
	return nil
}

// translate 将 repository 错误映射为业务错误码。
func translate(err error, conflictMsg string, args ...any) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return errcode.NotFound.WithMessage("example not found")
	case errors.Is(err, repository.ErrDuplicate):
		if conflictMsg != "" {
			return errcode.Conflict.WithMessage(conflictMsg, args...)
		}
		return errcode.Conflict
	default:
		return errcode.Internal.Wrap(err)
	}
}
