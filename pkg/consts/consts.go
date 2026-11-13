/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-12 07:45:31 America/Los_Angeles
 *****************************************************************************/

package consts

const (
	RequestTimestamp = "Kylin-Request-Timestamp" // gin.Context key：请求开始时间（time.Time）
	TraceHeader      = "Kylin-X-Trace-Id"        // 请求 / 响应头：链路 ID
	TraceKey         = "trace_id"                // gin.Context key / 日志字段：链路 ID
)
