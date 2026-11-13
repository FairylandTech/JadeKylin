/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-11 19:22:23 America/Los_Angeles
 *****************************************************************************/

package handler

import (
	"github.com/gin-gonic/gin"
	"jkylin.com/kylin/v2/internal/model"
	"jkylin.com/kylin/v2/internal/pkg/support"
	"jkylin.com/kylin/v2/internal/service"
)

// ExampleHandler 示例接口：参数绑定校验 -> service -> 统一响应。
type ExampleHandler struct {
	support.HandlerSupportBase
	svc *service.ExampleService
}

func NewExampleHandler(svc *service.ExampleService) *ExampleHandler {
	return &ExampleHandler{svc: svc}
}

// Example GET /example
func (h *ExampleHandler) Example(ctx *gin.Context) {
	h.Logger(ctx).Debug("example api endpoint")
	h.Success(ctx, map[string]string{"example": "Hello Gin Server."})
}

// Create POST /example/items
func (h *ExampleHandler) Create(ctx *gin.Context) {
	var req model.CreateExampleRequest
	if err := h.BindJSON(ctx, &req); err != nil {
		h.Fail(ctx, err)
		return
	}
	item, err := h.svc.Create(ctx.Request.Context(), &req)
	if err != nil {
		h.Fail(ctx, err)
		return
	}
	h.Success(ctx, item)
}

// Get GET /example/items/:id
func (h *ExampleHandler) Get(ctx *gin.Context) {
	var uri model.ExampleURI
	if err := h.BindURI(ctx, &uri); err != nil {
		h.Fail(ctx, err)
		return
	}
	item, err := h.svc.Get(ctx.Request.Context(), uri.ID)
	if err != nil {
		h.Fail(ctx, err)
		return
	}
	h.Success(ctx, item)
}

// List GET /example/items?page=1&size=10&keyword=xx
func (h *ExampleHandler) List(ctx *gin.Context) {
	var q model.ListExampleQuery
	if err := h.BindQuery(ctx, &q); err != nil {
		h.Fail(ctx, err)
		return
	}
	list, total, err := h.svc.List(ctx.Request.Context(), &q)
	if err != nil {
		h.Fail(ctx, err)
		return
	}
	h.Page(ctx, list, q.Page, q.Size, total)
}

// Update PUT /example/items/:id
func (h *ExampleHandler) Update(ctx *gin.Context) {
	var uri model.ExampleURI
	if err := h.BindURI(ctx, &uri); err != nil {
		h.Fail(ctx, err)
		return
	}
	var req model.UpdateExampleRequest
	if err := h.BindJSON(ctx, &req); err != nil {
		h.Fail(ctx, err)
		return
	}
	item, err := h.svc.Update(ctx.Request.Context(), uri.ID, &req)
	if err != nil {
		h.Fail(ctx, err)
		return
	}
	h.Success(ctx, item)
}

// Delete DELETE /example/items/:id
func (h *ExampleHandler) Delete(ctx *gin.Context) {
	var uri model.ExampleURI
	if err := h.BindURI(ctx, &uri); err != nil {
		h.Fail(ctx, err)
		return
	}
	if err := h.svc.Delete(ctx.Request.Context(), uri.ID); err != nil {
		h.Fail(ctx, err)
		return
	}
	h.Success(ctx, nil)
}

// Panic GET /example/panic —— 演示 Recovery 中间件，仅非 release 模式注册。
func (h *ExampleHandler) Panic(*gin.Context) {
	panic("example panic for recovery middleware demo")
}
