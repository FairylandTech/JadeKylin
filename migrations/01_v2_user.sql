-- CREATE TABLE IF NOT EXISTS users
-- (
--     id            BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
--     username      VARCHAR(191) NOT NULL UNIQUE,
--     display       VARCHAR(191) NOT NULL DEFAULT '',
--     note          LONGTEXT     NOT NULL DEFAULT '',
--     department_id BIGINT       NOT NULL DEFAULT 0,
--     status        VARCHAR(191) NOT NULL DEFAULT 'active',
--     roles_json    LONGTEXT     NOT NULL DEFAULT '[]',
--     pass_salt     VARCHAR(191) NOT NULL DEFAULT '',
--     pass_hash     VARCHAR(191) NOT NULL DEFAULT '',
--     created_at    DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
-- );

create table if not exists kylin_v2_users
(
    id         bigint generated always as identity primary key,
    username   varchar(128) not null unique,
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

-- CREATE TABLE IF NOT EXISTS roles (
--                                      id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
--                                      name VARCHAR(191) NOT NULL UNIQUE,
--                                      description LONGTEXT NOT NULL DEFAULT '',
--                                      perms_json LONGTEXT NOT NULL DEFAULT '[]'
-- );


