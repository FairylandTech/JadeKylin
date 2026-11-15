--------------------------------------------------------------------------------
-- Copyright (c) 2026 Beau Dean. All rights reserved.
--------------------------------------------------------------------------------

--------------------------------------------------------------------------------
-- @software: GoLand
-- @author: Beau Dean (Hean)
-- @contact: https://github.com/FairylandTech
-- @organization: https://github.com/FairylandFuture
-- @datetime: 2026-11-14 23:10:00 America/Los_Angeles
--------------------------------------------------------------------------------

-- 会话续期：会话存在时刷新空闲过期时间，可选替换保存的 JWT。
--
-- KEYS[1]  会话 Hash
-- KEYS[2]  用户会话索引 ZSET
-- ARGV[1]  空闲超时（毫秒）
-- ARGV[2]  续签后的 JWT；为空字符串时不更新
--
-- 会话存在返回 1，不存在返回 0。

local sessionKey = KEYS[1]
local sessionIndexKey = KEYS[2]
local idleTimeout = ARGV[1]
local renewedToken = ARGV[2]

if redis.call('EXISTS', sessionKey) == 0 then
  return 0
end
if renewedToken ~= '' then
  redis.call('HSET', sessionKey, 'token', renewedToken)
end
redis.call('PEXPIRE', sessionKey, idleTimeout)
redis.call('PEXPIRE', sessionIndexKey, idleTimeout)
return 1
