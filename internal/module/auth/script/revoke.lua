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

-- 让用户的所有会话下线：删除索引中的每个会话，再删除索引本身。
--
-- KEYS[1]  用户会话索引 ZSET
-- ARGV[1]  会话 Hash 的 key 前缀（拼上 session id 即会话 key）
--
-- 返回被删除的索引 key 数量（0 或 1）。

local sessionIndexKey = KEYS[1]
local sessionKeyPrefix = ARGV[1]

for _, member in ipairs(redis.call('ZRANGE', sessionIndexKey, 0, -1)) do
  redis.call('DEL', sessionKeyPrefix .. member)
end
return redis.call('DEL', sessionIndexKey)
