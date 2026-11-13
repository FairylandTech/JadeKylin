/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

package handler

import (
	"github.com/gin-gonic/gin"
	"jkylin.com/kylin/v2/internal/contracts/requests"
	"jkylin.com/kylin/v2/internal/pkg/support"
	user2 "jkylin.com/kylin/v2/internal/service"
)

type UserHandler struct {
	support.HandlerSupportBase
	userService *user2.UserService
}

func NewUserHandler(userService *user2.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// Create POST /users
func (handler *UserHandler) Create(ctx *gin.Context) {
	var requestBody requests.CreateRequest
	if err := handler.BindJSON(ctx, &requestBody); err != nil {
		handler.Fail(ctx, err)
		return
	}
	record, err := handler.userService.Create(ctx.Request.Context(), &requestBody)
	if err != nil {
		handler.Fail(ctx, err)
		return
	}
	handler.Success(ctx, record)
}

// Get GET /users/:id
func (handler *UserHandler) Get(ctx *gin.Context) {
	var uri requests.URI
	if err := handler.BindURI(ctx, &uri); err != nil {
		handler.Fail(ctx, err)
		return
	}
	record, err := handler.userService.Get(ctx.Request.Context(), uri.ID)
	if err != nil {
		handler.Fail(ctx, err)
		return
	}
	handler.Success(ctx, record)
}

// List GET /users?page=1&size=10&keyword=xx
func (handler *UserHandler) List(ctx *gin.Context) {
	var q requests.ListQuery
	if err := handler.BindQuery(ctx, &q); err != nil {
		handler.Fail(ctx, err)
		return
	}
	list, total, err := handler.userService.List(ctx.Request.Context(), &q)
	if err != nil {
		handler.Fail(ctx, err)
		return
	}
	handler.Page(ctx, list, q.Page, q.Size, total)
}

// Update PUT /users/:id
func (handler *UserHandler) Update(ctx *gin.Context) {
	var uri requests.URI
	if err := handler.BindURI(ctx, &uri); err != nil {
		handler.Fail(ctx, err)
		return
	}
	var req requests.UpdateRequest
	if err := handler.BindJSON(ctx, &req); err != nil {
		handler.Fail(ctx, err)
		return
	}
	record, err := handler.userService.Update(ctx.Request.Context(), uri.ID, &req)
	if err != nil {
		handler.Fail(ctx, err)
		return
	}
	handler.Success(ctx, record)
}

// Delete DELETE /users/:id
func (handler *UserHandler) Delete(ctx *gin.Context) {
	var uri requests.URI
	if err := handler.BindURI(ctx, &uri); err != nil {
		handler.Fail(ctx, err)
		return
	}
	if err := handler.userService.Delete(ctx.Request.Context(), uri.ID); err != nil {
		handler.Fail(ctx, err)
		return
	}
	handler.Success(ctx, nil)
}
