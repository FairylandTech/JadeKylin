--------------------------------------------------------------------------------
-- Copyright (c) 2026 Beau Dean. All rights reserved.
--------------------------------------------------------------------------------

--------------------------------------------------------------------------------
-- @software: GoLand
-- @author: Beau Dean (Hean)
-- @contact: https://github.com/FairylandTech
-- @organization: https://github.com/FairylandFuture
-- @datetime: 2026-11-15 10:00:00 America/Los_Angeles
--------------------------------------------------------------------------------

-- 权限判断：用户是否拥有某个权限 code，一次往返完成；命中时为用户权限集合续期。
--
-- KEYS[1]  用户权限 SET
-- ARGV[1]  需要的权限 code
-- ARGV[2]  用户权限集合的空闲过期时间（毫秒）
--
-- 返回：
--   -1  用户权限未缓存，需要从数据库加载
--    0  拒绝
--    1  放行

local userPermissionsKey = KEYS[1]
local permission = ARGV[1]
local userTimeToLive = ARGV[2]

if redis.call('EXISTS', userPermissionsKey) == 0 then
  return -1
end
redis.call('PEXPIRE', userPermissionsKey, userTimeToLive)

if permission ~= '' and redis.call('SISMEMBER', userPermissionsKey, permission) == 1 then
  return 1
end
return 0
