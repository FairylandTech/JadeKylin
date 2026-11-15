/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 23:30:00 America/Los_Angeles
 *****************************************************************************/

package requests

import "jkylin.com/kylin/v2/internal/model"

type LoginRequest struct {
	Username    string `json:"username" binding:"required,max=128"`
	Password    string `json:"password" binding:"required,max=72"`
	CaptchaID   string `json:"captcha_id" binding:"required,max=64"`
	CaptchaCode string `json:"captcha_code" binding:"required,len=4,numeric"`
}

type CaptchaResponse struct {
	CaptchaID string `json:"captcha_id"`
	Code      string `json:"code"`       // 4 位数字验证码明文
	ExpiresIn int    `json:"expires_in"` // 秒
}

type LoginResponse struct {
	AccessToken string      `json:"access_token"`
	TokenType   string      `json:"token_type"`
	ExpiresIn   int         `json:"expires_in"` // 秒
	User        *model.User `json:"user"`
}
