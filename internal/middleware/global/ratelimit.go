/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-11 21:48:49 America/Los_Angeles
 *****************************************************************************/

package global

import (
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"jkylin.com/kylin/v2/pkg/errcode"
	"jkylin.com/kylin/v2/pkg/response"
)

// RateLimitMiddleware 基于客户端 IP 的令牌桶限流（单机内存实现）。
type RateLimitMiddleware struct {
	rate    float64 // 每秒补充令牌数
	burst   float64 // 桶容量
	idleTTL time.Duration

	mu      sync.Mutex
	buckets map[string]*bucket
}

type bucket struct {
	tokens float64
	last   time.Time
}

func NewRateLimit(rps float64, burst int) *RateLimitMiddleware {
	if rps <= 0 {
		rps = 1
	}
	if burst <= 0 {
		burst = int(math.Ceil(rps))
	}
	rl := &RateLimitMiddleware{
		rate:    rps,
		burst:   float64(burst),
		idleTTL: 3 * time.Minute,
		buckets: make(map[string]*bucket),
	}
	go rl.cleanup()
	return rl
}

// Allow 判断 key 是否允许通过，并返回令牌不足时建议的等待时长。
func (rl *RateLimitMiddleware) Allow(key string) (bool, time.Duration) {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{tokens: rl.burst, last: now}
		rl.buckets[key] = b
	}
	b.tokens = math.Min(rl.burst, b.tokens+now.Sub(b.last).Seconds()*rl.rate)
	b.last = now

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	wait := time.Duration((1 - b.tokens) / rl.rate * float64(time.Second))
	return false, wait
}

// cleanup 定期清理长时间未访问的桶，防止内存无限增长。
func (rl *RateLimitMiddleware) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-rl.idleTTL)
		rl.mu.Lock()
		for k, b := range rl.buckets {
			if b.last.Before(cutoff) {
				delete(rl.buckets, k)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimitMiddleware) Handler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ok, wait := rl.Allow(ctx.ClientIP())
		if !ok {
			ctx.Header("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
			response.Abort(ctx, errcode.TooManyRequests)
			return
		}
		ctx.Next()
	}
}

// RateLimit 按配置创建限流中间件。
// TODO: 后期按需优化。
func (m *Middleware) RateLimit() gin.HandlerFunc {
	return NewRateLimit(m.cfg.RateLimit.RPS, m.cfg.RateLimit.Burst).Handler()
}
