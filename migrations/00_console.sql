/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

-- 使用 postgres 用户创建一个 kylin 用户
create user kylin with encrypted password '37761cefb8244f7ca9098379b0c4c257';
-- 创建数据库 kylin 并授权给 kylin 用户
create database kylin owner kylin;
-- 授权 kylin 用户对 kylin 数据库的所有权限
grant all privileges on database kylin to kylin;

SELECT pid,
       datname,
       usename,
       application_name,
       client_addr,
       state,
       backend_start,
       query
FROM pg_stat_activity
ORDER BY backend_start;
