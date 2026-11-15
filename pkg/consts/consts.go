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
	GinMode            = "GIN_MODE"
	RunMode            = "RUN_MODE" // local | dev 时跳过登录认证（见 config.App.SkipAuth）
	KylinDatabaseDsn   = "KYLIN_DATABASE_DSN"
	KylinRedisURL      = "KYLIN_REDIS_URL"
	KylinAdminUsername = "KYLIN_ADMIN_USERNAME" // 启动时自动创建的初始管理员用户名
	KylinAdminPassword = "KYLIN_ADMIN_PASSWORD" // 初始管理员密码，仅在该用户不存在时使用
	KylinJWTSecret     = "KYLIN_JWT_SECRET"

	RequestTimestamp  = "Kylin-Request-Timestamp" // 请求开始时间（用于计算处理耗时）
	TraceHeader       = "Kylin-X-Trace-Id"        // 响应头中返回的 Trace ID
	TraceKey          = "trace_id"                // 上下文中存储的 Trace ID 键
	AccessTokenHeader = "Kylin-Access-Token"      // 续签后返回新 JWT 的响应头
	UserIDKey         = "user_id"                 // gin 上下文中存储当前登录用户 id 的键
	EnvConfigPath     = "KYLIN_CONFIG"            // 指定配置文件路径的环境变量
	DefaultConfigPath = "configs/config.yaml"     // 默认配置文件路径
)
