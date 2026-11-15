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

-- 创建登录会话：清理已过期的会话，按上限踢掉最早登录的会话，然后写入新会话。
--
-- KEYS[1]  用户会话索引 ZSET（session id -> 登录时间）
-- KEYS[2]  新会话 Hash
-- ARGV[1]  会话 Hash 的 key 前缀（拼上 session id 即会话 key）
-- ARGV[2]  新会话 id
-- ARGV[3]  登录时间（毫秒时间戳）
-- ARGV[4]  空闲超时（毫秒）
-- ARGV[5]  同一用户允许的最大会话数，0 表示不限制
-- ARGV[6…] 会话 Hash 的 field / value
--
-- 返回被踢下线的 session id 列表。

local sessionIndexKey = KEYS[1]
local sessionKey = KEYS[2]
local sessionKeyPrefix = ARGV[1]
local sessionID = ARGV[2]
local loginAt = tonumber(ARGV[3])
local idleTimeout = tonumber(ARGV[4])
local maximumSessions = tonumber(ARGV[5])

for _, member in ipairs(redis.call('ZRANGE', sessionIndexKey, 0, -1)) do
  if redis.call('EXISTS', sessionKeyPrefix .. member) == 0 then
    redis.call('ZREM', sessionIndexKey, member)
  end
end

local evicted = {}
if maximumSessions > 0 then
  local count = redis.call('ZCARD', sessionIndexKey)
  if count >= maximumSessions then
    for _, member in ipairs(redis.call('ZRANGE', sessionIndexKey, 0, count - maximumSessions)) do
      redis.call('DEL', sessionKeyPrefix .. member)
      redis.call('ZREM', sessionIndexKey, member)
      table.insert(evicted, member)
    end
  end
end

redis.call('HSET', sessionKey, unpack(ARGV, 6))
redis.call('PEXPIRE', sessionKey, idleTimeout)
redis.call('ZADD', sessionIndexKey, loginAt, sessionID)
redis.call('PEXPIRE', sessionIndexKey, idleTimeout)
return evicted
