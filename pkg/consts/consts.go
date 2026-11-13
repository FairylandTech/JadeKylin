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
	GinMode             = "GIN_MODE"
	KylinDatabaseDsn    = "KYLIN_DATABASE_DSN"
	KylinRedisURL       = "KYLIN_REDIS_URL"
	KylinUserAdminToken = "KYLIN_USER_ADMIN_TOKEN"

	RequestTimestamp  = "Kylin-Request-Timestamp" // 请求开始时间（用于计算处理耗时）
	TraceHeader       = "Kylin-X-Trace-Id"        // 响应头中返回的 Trace ID
	TraceKey          = "trace_id"                // 上下文中存储的 Trace ID 键
	EnvConfigPath     = "KYLIN_CONFIG"            // 指定配置文件路径的环境变量
	DefaultConfigPath = "configs/config.yaml"     // 默认配置文件路径
)
