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

const (
	// bucketIdleTimeout 是令牌桶无访问后被清理前的保留时长。
	bucketIdleTimeout = 3 * time.Minute
	// bucketCleanupInterval 是清理空闲令牌桶的周期。
	bucketCleanupInterval = time.Minute
)

// RateLimitMiddleware 基于客户端 IP 的令牌桶限流（单机内存实现）。
type RateLimitMiddleware struct {
	rate        float64 // 每秒补充令牌数
	burst       float64 // 桶容量
	idleTimeout time.Duration

	mutex   sync.Mutex
	buckets map[string]*tokenBucket
}

type tokenBucket struct {
	tokens     float64
	lastAccess time.Time
}

func NewRateLimit(requestsPerSecond float64, burst int) *RateLimitMiddleware {
	if requestsPerSecond <= 0 {
		requestsPerSecond = 1
	}
	if burst <= 0 {
		burst = int(math.Ceil(requestsPerSecond))
	}
	rateLimiter := &RateLimitMiddleware{
		rate:        requestsPerSecond,
		burst:       float64(burst),
		idleTimeout: bucketIdleTimeout,
		buckets:     make(map[string]*tokenBucket),
	}
	go rateLimiter.cleanup()
	return rateLimiter
}

// Allow 判断 key 是否允许通过，并返回令牌不足时建议的等待时长。
func (rateLimiter *RateLimitMiddleware) Allow(key string) (bool, time.Duration) {
	now := time.Now()
	rateLimiter.mutex.Lock()
	defer rateLimiter.mutex.Unlock()

	bucket, ok := rateLimiter.buckets[key]
	if !ok {
		bucket = &tokenBucket{tokens: rateLimiter.burst, lastAccess: now}
		rateLimiter.buckets[key] = bucket
	}
	bucket.tokens = math.Min(rateLimiter.burst, bucket.tokens+now.Sub(bucket.lastAccess).Seconds()*rateLimiter.rate)
	bucket.lastAccess = now

	if bucket.tokens >= 1 {
		bucket.tokens--
		return true, 0
	}
	wait := time.Duration((1 - bucket.tokens) / rateLimiter.rate * float64(time.Second))
	return false, wait
}

// cleanup 定期清理长时间未访问的桶，防止内存无限增长。
func (rateLimiter *RateLimitMiddleware) cleanup() {
	ticker := time.NewTicker(bucketCleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-rateLimiter.idleTimeout)
		rateLimiter.mutex.Lock()
		for key, bucket := range rateLimiter.buckets {
			if bucket.lastAccess.Before(cutoff) {
				delete(rateLimiter.buckets, key)
			}
		}
		rateLimiter.mutex.Unlock()
	}
}

func (rateLimiter *RateLimitMiddleware) Handler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ok, wait := rateLimiter.Allow(ctx.ClientIP())
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
func (globalMiddleware *Middleware) RateLimit() gin.HandlerFunc {
	return NewRateLimit(globalMiddleware.configuration.RateLimit.RPS, globalMiddleware.configuration.RateLimit.Burst).Handler()
}
