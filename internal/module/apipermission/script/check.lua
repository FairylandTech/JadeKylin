--------------------------------------------------------------------------------
-- Copyright (c) 2026 Beau Dean. All rights reserved.
--------------------------------------------------------------------------------

--------------------------------------------------------------------------------
-- @software: GoLand
-- @author: Beau Dean (Hean)
-- @contact: https://github.com/FairylandTech
-- @organization: https://github.com/FairylandFuture
-- @datetime: 2026-11-15 02:30:00 America/Los_Angeles
--------------------------------------------------------------------------------

-- 接口鉴权：判断用户能否访问接口，一次往返完成；命中时为用户权限集合续期。
--
-- KEYS[1]  接口放行权限 HASH（"METHOD path" -> permission）
-- KEYS[2]  用户权限 SET
-- ARGV[1]  接口标识 "METHOD path"
-- ARGV[2]  用户权限集合的空闲过期时间（毫秒）
--
-- 返回：
--   -1  接口权限表未缓存，需要从数据库同步
--   -2  用户权限未缓存，需要从数据库加载
--    0  接口未登记放行权限
--    1  拒绝
--    2  放行

local apiPermissionsKey = KEYS[1]
local userPermissionsKey = KEYS[2]
local endpoint = ARGV[1]
local userTimeToLive = ARGV[2]

if redis.call('EXISTS', apiPermissionsKey) == 0 then
  return -1
end
if redis.call('EXISTS', userPermissionsKey) == 0 then
  return -2
end
redis.call('PEXPIRE', userPermissionsKey, userTimeToLive)

local permission = redis.call('HGET', apiPermissionsKey, endpoint)
if not permission or endpoint == '' then
  return 0
end
if redis.call('SISMEMBER', userPermissionsKey, permission) == 1 then
  return 2
end
return 1
