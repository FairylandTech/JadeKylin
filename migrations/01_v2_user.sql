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
        '{"dashboard.view","console.view","console.command","console.settings","console.maintenance","research.view","research.export","research.purge","themes.install","themes.policy","logs.view","logs.clear","settings.write","system.users","system.roles"}'),
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
       ('system.roles', '管理角色', '系统治理')
on conflict do nothing;
