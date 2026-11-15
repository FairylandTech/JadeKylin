create extension if not exists pg_trgm;

-- Users
create table if not exists kylin_v2_users
(
    id         bigint generated always as identity primary key,
    username   varchar(128) not null,
    password   varchar(128) not null,
    nickname   varchar(128),
    remark     varchar(512),
    status     varchar(1)   not null default 'Y',
    roles      varchar[]    not null default '{}',
    created_at timestamptz  not null default current_timestamp,
    updated_at timestamptz  not null default current_timestamp,
    enabled    varchar(1)   not null default 'Y',
    version    integer      not null default 1
);
create unique index if not exists uk_kylin_v2_users_username on kylin_v2_users (username) where enabled = 'Y';
create index if not exists idx_kylin_v2_users_username_trgm on kylin_v2_users using gin (username gin_trgm_ops) where enabled = 'Y';
create index if not exists idx_kylin_v2_users_nickname_trgm on kylin_v2_users using gin (nickname gin_trgm_ops) where enabled = 'Y';

-- Roles
create table if not exists kylin_v2_roles
(
    id          bigint generated always as identity primary key,
    code        varchar(128) not null,
    name        varchar(128) not null,
    description varchar(512),
    permissions varchar[]    not null default '{}',
    created_at  timestamptz  not null default current_timestamp,
    updated_at  timestamptz  not null default current_timestamp,
    enabled     varchar(1)   not null default 'Y'
);
create unique index if not exists uk_kylin_v2_roles_code on kylin_v2_roles (code) where enabled = 'Y';
create unique index if not exists uk_kylin_v2_roles_name on kylin_v2_roles (name) where enabled = 'Y';

-- Permissions
create table if not exists kylin_v2_permissions
(
    id          bigint generated always as identity primary key,
    code        varchar(128) not null,
    name        varchar(128) not null,
    category    varchar(128) not null,
    description varchar(512),
    created_at  timestamptz  not null default current_timestamp,
    updated_at  timestamptz  not null default current_timestamp,
    enabled     varchar(1)   not null default 'Y'
);
create unique index if not exists uk_kylin_v2_permissions_code on kylin_v2_permissions (code) where enabled = 'Y';
create unique index if not exists uk_kylin_v2_permissions_name on kylin_v2_permissions (name) where enabled = 'Y';

insert into kylin_v2_roles (code, name, description, permissions)
values ('admin', '超级管理员', '全部权限',
        '{"system.apis","dashboard.view","console.view","console.command","console.settings","console.maintenance","research.view","research.export","research.purge","themes.install","themes.policy","logs.view","logs.clear","settings.write","system.users","system.roles"}'),
       ('operator', '运营操作员', '实时控制台 + 资料中心',
        '{"console.view","console.command","console.settings","console.maintenance","research.view","research.export"}'),
       ('auditor', '安全审计', '只读审计', '{"logs.view","research.view"}')
on conflict do nothing;

insert into kylin_v2_permissions (code, name, category)
values ('dashboard.view', '查看数据概览', '数据概览'),
       ('console.view', '查看控制台', '实时控制台'),
       ('console.command', '下发指令', '实时控制台'),
       ('console.settings', '调整控制台设置', '实时控制台'),
       ('console.maintenance', '维护访客状态', '实时控制台'),
       ('research.view', '查询资料', '资料中心'),
       ('research.export', '导出资料', '资料中心'),
       ('research.purge', '清空资料', '资料中心'),
       ('themes.install', '安装主题', '主题与策略'),
       ('themes.policy', '编辑访问策略', '主题与策略'),
       ('logs.view', '查看日志', '运维'),
       ('logs.clear', '清空日志', '运维'),
       ('settings.write', '修改通用设置', '运维'),
       ('system.users', '管理账号', '系统治理'),
       ('system.roles', '管理角色', '系统治理'),
       ('system.apis', '管理接口权限', '系统治理')
on conflict do nothing;

-- 接口放行权限：每个受保护接口（method + gin 路由模式）对应一个权限 code。
-- 未登记的接口一律拒绝访问（HTTP 403）。可通过 /api-permissions 接口直接修改，修改后立即生效。
create table if not exists kylin_v2_api_permissions
(
    id          bigint generated always as identity primary key,
    method      varchar(16)  not null,
    path        varchar(256) not null,
    permission  varchar(128) not null,
    description varchar(512),
    created_at  timestamptz  not null default current_timestamp,
    updated_at  timestamptz  not null default current_timestamp,
    enabled     varchar(1)   not null default 'Y'
);

create unique index if not exists uk_kylin_v2_api_permissions_route on kylin_v2_api_permissions (method, path) where enabled = 'Y';

insert into kylin_v2_api_permissions (method, path, permission, description)
values ('POST', '/users', 'system.users', '创建用户'),
       ('GET', '/users', 'system.users', '分页查询用户'),
       ('GET', '/users/:id', 'system.users', '查询用户'),
       ('PUT', '/users/:id', 'system.users', '修改用户'),
       ('DELETE', '/users/:id', 'system.users', '删除用户'),
       ('PUT', '/users/:id/password', 'system.users', '修改用户密码'),
       ('POST', '/users/:id/password/reset', 'system.users', '重置用户密码'),
       ('POST', '/roles', 'system.roles', '创建角色'),
       ('GET', '/roles', 'system.roles', '分页查询角色'),
       ('GET', '/roles/:id', 'system.roles', '查询角色'),
       ('PUT', '/roles/:id', 'system.roles', '修改角色'),
       ('DELETE', '/roles/:id', 'system.roles', '删除角色'),
       ('GET', '/permissions', 'system.roles', '查询全部权限'),
       ('GET', '/api-permissions', 'system.apis', '查询接口放行权限'),
       ('PUT', '/api-permissions/:id', 'system.apis', '修改接口放行权限')
on conflict do nothing;
