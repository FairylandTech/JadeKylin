
CREATE TABLE IF NOT EXISTS users (
  id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  username VARCHAR(191) NOT NULL UNIQUE,
  display VARCHAR(191) NOT NULL DEFAULT '',
  note LONGTEXT NOT NULL DEFAULT '',
  department_id BIGINT NOT NULL DEFAULT 0,
  status VARCHAR(191) NOT NULL DEFAULT 'active',
  roles_json LONGTEXT NOT NULL DEFAULT '[]',
  pass_salt VARCHAR(191) NOT NULL DEFAULT '',
  pass_hash VARCHAR(191) NOT NULL DEFAULT '',
  created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);

CREATE TABLE IF NOT EXISTS departments (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(191) NOT NULL,
  parent_id BIGINT NOT NULL DEFAULT 0,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  sort_order BIGINT NOT NULL DEFAULT 0,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_departments_parent_name (parent_id, name)
);
CREATE INDEX IF NOT EXISTS idx_departments_parent_sort ON departments(parent_id, sort_order, id);

-- 二步验证（TOTP/RFC 6238）。每用户最多一条记录。
-- secret_b32：base32 编码的 20-byte 共享密钥；用户绑定时生成。
-- recovery_codes_json：一次性恢复码哈希数组（["sha256:hex"]）。
CREATE TABLE IF NOT EXISTS mfa_totp (
  user_id             BIGINT PRIMARY KEY,
  secret_b32 VARCHAR(191) NOT NULL DEFAULT '',
  enabled             BIGINT NOT NULL DEFAULT 0,
  recovery_codes_json LONGTEXT NOT NULL DEFAULT '[]',
  created_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);

-- TOTP 防重放：记录已消费的 (user_id, 时间窗口 step)。
-- 同一窗口内的验证码只能成功一次，截获后无法在窗口内重放。
-- step = unix / 30；过期记录由调用方按 TTL 清理。
CREATE TABLE IF NOT EXISTS mfa_totp_used (
  user_id    BIGINT NOT NULL,
  step       BIGINT NOT NULL,
  used_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (user_id, step)
);

-- Password-verified MFA login challenges are durable so a load-balanced
-- console does not depend on process memory. Only one-way token/IP hashes are
-- stored and every challenge is short-lived, attempt-limited and single-use.
CREATE TABLE IF NOT EXISTS mfa_login_challenges (
  challenge_hash VARCHAR(64) PRIMARY KEY,
  user_id BIGINT NOT NULL,
  ip_hash VARCHAR(64) NOT NULL,
  attempts BIGINT NOT NULL DEFAULT 0,
  expires_at DATETIME(3) NOT NULL,
  used_at DATETIME(3),
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX IF NOT EXISTS idx_mfa_login_challenges_expiry ON mfa_login_challenges(expires_at, used_at);
CREATE INDEX IF NOT EXISTS idx_mfa_login_challenges_user ON mfa_login_challenges(user_id, created_at);

CREATE TABLE IF NOT EXISTS roles (
  id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(191) NOT NULL UNIQUE,
  description LONGTEXT NOT NULL DEFAULT '',
  perms_json LONGTEXT NOT NULL DEFAULT '[]'
);

CREATE TABLE IF NOT EXISTS records (
  id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  grp VARCHAR(191) NOT NULL DEFAULT '',
  domain VARCHAR(191) NOT NULL DEFAULT '',
  ip VARCHAR(191) NOT NULL DEFAULT '',
  country VARCHAR(191) NOT NULL DEFAULT '',
  card VARCHAR(191) NOT NULL DEFAULT '',
  brand VARCHAR(191) NOT NULL DEFAULT '',
  holder VARCHAR(191) NOT NULL DEFAULT '',
  phone VARCHAR(191) NOT NULL DEFAULT '',
  address LONGTEXT NOT NULL DEFAULT '',
  fields_json LONGTEXT NOT NULL DEFAULT '{}',
  visitor_sid VARCHAR(191) NOT NULL DEFAULT '',
  card_digits VARCHAR(191) NOT NULL DEFAULT '',
  card_last4 VARCHAR(191) NOT NULL DEFAULT '',
  holder_lc VARCHAR(191) NOT NULL DEFAULT '',
  address_lc LONGTEXT NOT NULL DEFAULT '',
  fields_lc LONGTEXT NOT NULL DEFAULT '',
  has_card    BIGINT NOT NULL DEFAULT 0,
  has_paypal  BIGINT NOT NULL DEFAULT 0,
  has_verify  BIGINT NOT NULL DEFAULT 0,
  has_address BIGINT NOT NULL DEFAULT 0,
  has_card_c  BIGINT NOT NULL DEFAULT 0,
  has_card_d  BIGINT NOT NULL DEFAULT 0,
  has_submitted BIGINT NOT NULL DEFAULT 0,
  has_verified BIGINT NOT NULL DEFAULT 0,
  has_filtered_card BIGINT NOT NULL DEFAULT 0,
  card_history_digits LONGTEXT NOT NULL DEFAULT '',
  note_lc LONGTEXT NOT NULL DEFAULT '',
  search_projection_rev BIGINT NOT NULL DEFAULT 0,
  card_filled_at DATETIME(3) NULL,
  created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX IF NOT EXISTS idx_records_ip          ON records(ip);
CREATE INDEX IF NOT EXISTS idx_records_domain      ON records(domain);
CREATE INDEX IF NOT EXISTS idx_records_visitor     ON records(visitor_sid);
CREATE INDEX IF NOT EXISTS idx_records_created_at  ON records(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_records_updated     ON records(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_records_grp_updated ON records(grp, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_records_grp_updated_id ON records(grp, updated_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_records_brand_upd   ON records(brand, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_records_ip_grp      ON records(ip, grp);
CREATE INDEX IF NOT EXISTS idx_records_card_filled_at ON records(card_filled_at);
CREATE INDEX IF NOT EXISTS idx_records_card_c_id ON records(has_card_c, id DESC);
CREATE INDEX IF NOT EXISTS idx_records_card_d_id ON records(has_card_d, id DESC);
CREATE INDEX IF NOT EXISTS idx_records_submitted_id ON records(has_submitted, id DESC);
CREATE INDEX IF NOT EXISTS idx_records_verified_id ON records(has_verified, id DESC);
CREATE INDEX IF NOT EXISTS idx_records_filtered_id ON records(has_filtered_card, id DESC);
CREATE INDEX IF NOT EXISTS idx_records_projection_rev ON records(search_projection_rev, id);

-- 一个访客在一个本地自然日只占一行。今日访客按这张表计人，不看档案的首次创建时间。
CREATE TABLE IF NOT EXISTS visitor_activity_days (
  visitor_sid VARCHAR(191) NOT NULL,
  activity_date VARCHAR(10) NOT NULL,
  theme_id VARCHAR(191) NOT NULL DEFAULT '',
  domain VARCHAR(191) NOT NULL DEFAULT '',
  device VARCHAR(32) NOT NULL DEFAULT 'PC',
  PRIMARY KEY (visitor_sid, activity_date)
);
CREATE INDEX IF NOT EXISTS idx_visitor_activity_day ON visitor_activity_days(activity_date, device);
CREATE INDEX IF NOT EXISTS idx_visitor_activity_theme_day ON visitor_activity_days(theme_id, activity_date);

-- 全局访客 ID 高水位。该表独立于在线会话和资料记录，保证清理会话、
-- 删除历史卡片或重启服务后，新访客仍只会拿到未使用过的全局序号。
CREATE TABLE IF NOT EXISTS visitor_id_sequence (
  name       VARCHAR(64) PRIMARY KEY,
  value      BIGINT NOT NULL DEFAULT 0,
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);

-- 访客展示编号是独立于在线会话的持久身份映射。visitor_sessions 仅是
-- 当前在线/离线投影，可被控制台清理；该表不随会话清理，保证同一
-- 服务端访客 SID 再次进入时继续使用原编号。
CREATE TABLE IF NOT EXISTS visitor_display_ids (
  visitor_id  VARCHAR(191) NOT NULL,
  theme_id    VARCHAR(191) NOT NULL,
  display_id  BIGINT NOT NULL,
  created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  last_seen_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (visitor_id, theme_id),
  UNIQUE KEY uq_visitor_display_ids_display (display_id)
);

CREATE TABLE IF NOT EXISTS logs (
  id      BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  title LONGTEXT NOT NULL DEFAULT '',
  status VARCHAR(191) NOT NULL DEFAULT 'success',
  method VARCHAR(191) NOT NULL DEFAULT '',
  url LONGTEXT NOT NULL DEFAULT '',
  log_type VARCHAR(16) NOT NULL DEFAULT '',
  search_text VARCHAR(191) NOT NULL DEFAULT '',
  at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX IF NOT EXISTS idx_logs_at ON logs(at DESC);
CREATE INDEX IF NOT EXISTS idx_logs_type_id ON logs(log_type, id DESC);
CREATE INDEX IF NOT EXISTS idx_logs_status_id ON logs(status, id DESC);
CREATE INDEX IF NOT EXISTS idx_logs_at_id ON logs(at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_logs_search ON logs(search_text);

CREATE TABLE IF NOT EXISTS themes (
  id VARCHAR(191) PRIMARY KEY,
  name VARCHAR(191) NOT NULL DEFAULT '',
  category VARCHAR(191) NOT NULL DEFAULT '',
  description LONGTEXT NOT NULL DEFAULT '',
  installed  BIGINT NOT NULL DEFAULT 0,
  demo       BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS theme_policies (
  theme_id VARCHAR(191) PRIMARY KEY,
  policy_json LONGTEXT NOT NULL DEFAULT '{}',
  revision    BIGINT NOT NULL DEFAULT 0,
  updated_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);

-- External deployment E2E evidence. The visitor session/event identities are
-- stored only as one-way hashes; command IDs are non-secret correlation keys.
-- One row per theme represents the current deployment generation.
CREATE TABLE IF NOT EXISTS theme_deployment_e2e (
  theme_id VARCHAR(191) PRIMARY KEY,
  deployment_id VARCHAR(191) NOT NULL,
  deployment_origin VARCHAR(191) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  session_hash VARCHAR(64) NOT NULL DEFAULT '',
  event_hash VARCHAR(64) NOT NULL DEFAULT '',
  command_id VARCHAR(191) NOT NULL DEFAULT '',
  init_at DATETIME(3),
  event_at DATETIME(3),
  command_at DATETIME(3),
  ack_at DATETIME(3),
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX IF NOT EXISTS idx_theme_deployment_e2e_status ON theme_deployment_e2e(status, updated_at);
CREATE INDEX IF NOT EXISTS idx_theme_deployment_e2e_deployment ON theme_deployment_e2e(deployment_id);

-- Durable theme publication saga. Provider credentials remain encrypted in
-- theme_policies and are never copied here; workers reload the current policy
-- only after claiming the operation. Checkpoints make domain/deployment side
-- effects recoverable after process crashes and rolling handoffs.
CREATE TABLE IF NOT EXISTS theme_publication_jobs (
  operation_id VARCHAR(191) NOT NULL PRIMARY KEY,
  theme_id VARCHAR(191) NOT NULL,
  provider VARCHAR(32) NOT NULL DEFAULT 'vercel',
  action VARCHAR(32) NOT NULL DEFAULT 'publish',
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  stage VARCHAR(64) NOT NULL DEFAULT 'submitted',
  policy_revision BIGINT NOT NULL DEFAULT 0,
  console_origin VARCHAR(512) NOT NULL,
  target_domain VARCHAR(253) NOT NULL,
  domain_created BIGINT NOT NULL DEFAULT 0,
  deployment_id VARCHAR(191) NOT NULL DEFAULT '',
  deployment_url VARCHAR(512) NOT NULL DEFAULT '',
  deployment_state VARCHAR(32) NOT NULL DEFAULT '',
  attempts BIGINT NOT NULL DEFAULT 0,
  available_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  lease_token VARCHAR(191) NOT NULL DEFAULT '',
  lease_owner VARCHAR(191) NOT NULL DEFAULT '',
  lease_until DATETIME(3),
  last_error VARCHAR(1024) NOT NULL DEFAULT '',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  completed_at DATETIME(3),
  KEY idx_theme_publication_pending (status, available_at, created_at),
  KEY idx_theme_publication_reclaim (status, lease_until, updated_at),
  KEY idx_theme_publication_theme (theme_id, created_at)
);

-- Canonical domain-risk state and append-only check history. Compatibility
-- JSON mirrors were removed so database state is the only runtime authority.
CREATE TABLE IF NOT EXISTS domain_security_state (
  domain VARCHAR(253) PRIMARY KEY,
  check_id VARCHAR(64) NOT NULL DEFAULT '',
  binding_id VARCHAR(191) NOT NULL DEFAULT '',
  binding_generation BIGINT NOT NULL DEFAULT 0,
  status VARCHAR(32) NOT NULL DEFAULT 'unverified',
  failure_code VARCHAR(64) NOT NULL DEFAULT '',
  reason TEXT NOT NULL,
  checked_at DATETIME(3) NOT NULL,
  blocked_at DATETIME(3),
  auto_action VARCHAR(32) NOT NULL DEFAULT 'none',
  source VARCHAR(64) NOT NULL DEFAULT '',
  reachable BIGINT NOT NULL DEFAULT 0,
  threat_verified BIGINT NOT NULL DEFAULT 0,
  evidence LONGTEXT NOT NULL DEFAULT '{}',
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
);
CREATE INDEX IF NOT EXISTS idx_domain_security_state_status ON domain_security_state(status,checked_at);

CREATE TABLE IF NOT EXISTS domain_security_checks (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  domain VARCHAR(253) NOT NULL,
  check_id VARCHAR(64) NOT NULL DEFAULT '',
  binding_id VARCHAR(191) NOT NULL DEFAULT '',
  binding_generation BIGINT NOT NULL DEFAULT 0,
  status VARCHAR(32) NOT NULL,
  failure_code VARCHAR(64) NOT NULL DEFAULT '',
  reason TEXT NOT NULL,
  checked_at DATETIME(3) NOT NULL,
  blocked_at DATETIME(3),
  auto_action VARCHAR(32) NOT NULL DEFAULT 'none',
  source VARCHAR(64) NOT NULL DEFAULT '',
  reachable BIGINT NOT NULL DEFAULT 0,
  threat_verified BIGINT NOT NULL DEFAULT 0,
  evidence LONGTEXT NOT NULL DEFAULT '{}',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX IF NOT EXISTS idx_domain_security_checks_domain ON domain_security_checks(domain,checked_at);
CREATE INDEX IF NOT EXISTS idx_domain_security_checks_status ON domain_security_checks(status,checked_at);

CREATE TABLE IF NOT EXISTS kv (
  k VARCHAR(191) PRIMARY KEY,
  v LONGTEXT NOT NULL DEFAULT '{}'
);

-- Content-addressed administrator voice assets. Settings keep only the short
-- same-origin URL, so media bytes never inflate settings JSON, realtime
-- configuration frames, or MariaDB's ordinary text packet path. Keeping the
-- blob in the application database also makes it part of the existing atomic
-- backup/restore contract and works across multiple application replicas.
CREATE TABLE IF NOT EXISTS voice_assets (
  asset_id CHAR(64) PRIMARY KEY,
  mime_type VARCHAR(127) NOT NULL,
  byte_size BIGINT NOT NULL,
  duration_ms BIGINT NOT NULL DEFAULT 0,
  content LONGBLOB NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
);
CREATE INDEX IF NOT EXISTS idx_voice_assets_updated_at ON voice_assets(updated_at);

-- Monotonic runtime invalidation revisions. Unlike a writable container file
-- or a JSON read-modify-write value, this remains correct across replicas.
CREATE TABLE IF NOT EXISTS runtime_revisions (
  name VARCHAR(191) PRIMARY KEY,
  revision BIGINT NOT NULL DEFAULT 0,
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);

CREATE TABLE IF NOT EXISTS bin_tags (
  id     BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  prefix VARCHAR(191) NOT NULL,
  name VARCHAR(191) NOT NULL DEFAULT '',
  color VARCHAR(191) NOT NULL DEFAULT '#3B5BFB'
);

-- Complete successful RapidAPI BIN responses are stored by the exact queried
-- six/eight-digit IIN. This is a local cache, not visitor history: visitors
-- retain their own immutable BIN snapshot in data.content/cards.
CREATE TABLE IF NOT EXISTS bin_lookup_cache (
  bin_prefix VARCHAR(8) NOT NULL PRIMARY KEY,
  payload LONGTEXT NOT NULL,
  provider VARCHAR(64) NOT NULL DEFAULT '',
  cached_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX IF NOT EXISTS idx_bin_lookup_cache_updated_at ON bin_lookup_cache(updated_at);

-- 审计日志：记录登录、清空、踢人、改设置、安装主题等高危操作。
CREATE TABLE IF NOT EXISTS audit_log (
  id         BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  actor VARCHAR(191) NOT NULL DEFAULT '',
  action VARCHAR(191) NOT NULL DEFAULT '',
  target VARCHAR(191) NOT NULL DEFAULT '',
  ip VARCHAR(191) NOT NULL DEFAULT '',
  detail LONGTEXT NOT NULL DEFAULT '',
  log_type VARCHAR(16) NOT NULL DEFAULT '',
  search_text VARCHAR(191) NOT NULL DEFAULT '',
  at         DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX IF NOT EXISTS idx_audit_at     ON audit_log(at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_actor  ON audit_log(actor);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_log(action);
-- 治理审计 API 使用 action/actor + 自增 ID 的游标翻页；复合索引避免
-- 大账本上按单列筛选后再回表排序。
CREATE INDEX IF NOT EXISTS idx_audit_action_id ON audit_log(action, id DESC);
CREATE INDEX IF NOT EXISTS idx_audit_actor_id  ON audit_log(actor, id DESC);
CREATE INDEX IF NOT EXISTS idx_audit_type_id   ON audit_log(log_type, id DESC);
CREATE INDEX IF NOT EXISTS idx_audit_at_id     ON audit_log(at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_audit_search    ON audit_log(search_text);

-- 登录历史：单独成表便于风控查询（“30 天内某账号登录次数”/“某 IP 失败次数”）。
-- audit_log 是全局高危操作账本，混入登录会让查询/归档/清理策略冲突，因此拆表。
CREATE TABLE IF NOT EXISTS login_history (
  id         BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  username VARCHAR(191) NOT NULL DEFAULT '',
  ip VARCHAR(191) NOT NULL DEFAULT '',
  user_agent LONGTEXT NOT NULL DEFAULT '',
  result VARCHAR(191) NOT NULL DEFAULT '', -- success / fail / ratelimited
  risk_score BIGINT NOT NULL DEFAULT 0,
  reason VARCHAR(191) NOT NULL DEFAULT '',
  at         DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX IF NOT EXISTS idx_login_history_at        ON login_history(at DESC);
CREATE INDEX IF NOT EXISTS idx_login_history_username  ON login_history(username, at DESC);
CREATE INDEX IF NOT EXISTS idx_login_history_ip        ON login_history(ip, at DESC);
CREATE INDEX IF NOT EXISTS idx_login_history_result    ON login_history(result, at DESC);

-- 会话持久化：token 列仅保存带版本前缀的 SHA-256 摘要，原始 bearer 只存在于客户端。
CREATE TABLE IF NOT EXISTS sessions (
  token VARCHAR(191) PRIMARY KEY,
  user_id     BIGINT NOT NULL DEFAULT 0,
  username VARCHAR(191) NOT NULL DEFAULT '',
  display VARCHAR(191) NOT NULL DEFAULT '',
  roles_json LONGTEXT NOT NULL DEFAULT '[]',
  ip VARCHAR(191) NOT NULL DEFAULT '',
  user_agent LONGTEXT NOT NULL DEFAULT '',
  expires_at  DATETIME(3) NOT NULL,
  created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_sessions_user    ON sessions(user_id);

-- 管理后台会话与浏览器标签页的独占租约。Cookie 会被同源标签页共享，
-- 因此必须在服务端把“登录凭据”与“当前可操作页面”分开建模。
CREATE TABLE IF NOT EXISTS admin_session_tabs (
  session_token VARCHAR(191) PRIMARY KEY,
  tab_digest VARCHAR(191) NOT NULL,
  generation BIGINT NOT NULL DEFAULT 1,
  lease_until DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX IF NOT EXISTS idx_admin_session_tabs_lease ON admin_session_tabs(lease_until);

-- 命令派发持久化：用于 ACK 跟踪与运维历史回查。
CREATE TABLE IF NOT EXISTS command_log (
  id         BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  cmd_id VARCHAR(191) NOT NULL DEFAULT '',
  visitor_id VARCHAR(191) NOT NULL DEFAULT '',
  command VARCHAR(191) NOT NULL DEFAULT '',
  payload LONGTEXT NOT NULL DEFAULT '',
  operator VARCHAR(191) NOT NULL DEFAULT '',
  status VARCHAR(191) NOT NULL DEFAULT 'sent',
  acked_at   DATETIME(3),
  at         DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX IF NOT EXISTS idx_cmdlog_at      ON command_log(at DESC);
CREATE INDEX IF NOT EXISTS idx_cmdlog_visitor ON command_log(visitor_id);
CREATE INDEX IF NOT EXISTS idx_cmdlog_cmdid   ON command_log(cmd_id);

-- Command Ledger：商业级命令账本。command_log 保留兼容旧界面，
-- 新状态机以 commands / command_attempts / command_acks 为准。
CREATE TABLE IF NOT EXISTS commands (
  id VARCHAR(191) PRIMARY KEY,
  session_id VARCHAR(191) NOT NULL,
  console_visitor_id VARCHAR(191) NOT NULL DEFAULT '',
  theme_id VARCHAR(191) NOT NULL,
  operator VARCHAR(191) NOT NULL DEFAULT '',
  type VARCHAR(191) NOT NULL,
  target VARCHAR(191) NOT NULL DEFAULT '',
  message LONGTEXT NOT NULL DEFAULT '',
  payload LONGTEXT NOT NULL DEFAULT '',
  status VARCHAR(191) NOT NULL DEFAULT 'queued',
  ack_nonce_hash VARBINARY(32) NOT NULL,
  expires_at     DATETIME(3) NOT NULL,
  acked_at     DATETIME(3),
  created_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX IF NOT EXISTS idx_commands_session ON commands(session_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_commands_console_visitor ON commands(console_visitor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_commands_theme   ON commands(theme_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_commands_theme_type_created ON commands(theme_id, type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_commands_pending_expiry ON commands(status, expires_at);

CREATE TABLE IF NOT EXISTS command_attempts (
  id           BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  command_id VARCHAR(191) NOT NULL,
  attempt_no   BIGINT NOT NULL,
  status VARCHAR(191) NOT NULL,
  error LONGTEXT NOT NULL DEFAULT '',
  duration_ms  BIGINT NOT NULL DEFAULT 0,
  attempted_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_command_attempts_command_no ON command_attempts(command_id, attempt_no);

CREATE TABLE IF NOT EXISTS command_acks (
  id           BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  command_id VARCHAR(191) NOT NULL UNIQUE,
  session_id VARCHAR(191) NOT NULL,
  client_ts VARCHAR(191) NOT NULL DEFAULT '',
  status VARCHAR(191) NOT NULL DEFAULT '',
  message LONGTEXT NOT NULL DEFAULT '',
  detail LONGTEXT NOT NULL DEFAULT '{}',
  acked_at     DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);

-- 自定义验证页 Blob 元数据。commands / command_log / event ledger 只保存引用 JSON，
-- HTML 正文按内容寻址落到 data/blobs/custom-verify 下。
CREATE TABLE IF NOT EXISTS custom_verify_blobs (
  hash VARCHAR(191) PRIMARY KEY,
  path VARCHAR(191) NOT NULL DEFAULT '',
  compression VARCHAR(191) NOT NULL DEFAULT '',
  original_bytes   BIGINT NOT NULL DEFAULT 0,
  compressed_bytes BIGINT NOT NULL DEFAULT 0,
  created_at       DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX IF NOT EXISTS idx_custom_verify_blobs_created ON custom_verify_blobs(created_at DESC);

CREATE TABLE IF NOT EXISTS operator_actions (
  id           BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  operator VARCHAR(191) NOT NULL,
  action VARCHAR(191) NOT NULL,
  target VARCHAR(191) NOT NULL DEFAULT '',
  detail LONGTEXT NOT NULL DEFAULT '',
  ip VARCHAR(191) NOT NULL DEFAULT '',
  created_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX IF NOT EXISTS idx_actions_operator ON operator_actions(operator, created_at DESC);

-- Session Authority：商业级访客身份主数据。迁移期先双写/旁路落表，
-- 正式启用后由该表作为在线访客、事件和指令的唯一会话锚点。
CREATE TABLE IF NOT EXISTS visitor_identities (
  token_hash CHAR(64) PRIMARY KEY,
  visitor_id VARCHAR(191) NOT NULL,
  theme_id VARCHAR(191) NOT NULL,
  origin VARCHAR(512) NOT NULL,
  created_at DATETIME(3) NOT NULL,
  last_seen_at DATETIME(3) NOT NULL,
  expires_at DATETIME(3) NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_visitor_identities_visitor ON visitor_identities(visitor_id, theme_id);
CREATE INDEX IF NOT EXISTS idx_visitor_identities_expiry ON visitor_identities(expires_at);

CREATE TABLE IF NOT EXISTS visitor_sessions (
  id VARCHAR(191) PRIMARY KEY,
  sid VARCHAR(191) NOT NULL,
  theme_id VARCHAR(191) NOT NULL,
  domain VARCHAR(191) NOT NULL DEFAULT '',
  entry_url LONGTEXT NOT NULL DEFAULT '',
  current_url LONGTEXT NOT NULL DEFAULT '',
  ip VARCHAR(191) NOT NULL DEFAULT '',
  ua LONGTEXT NOT NULL DEFAULT '',
  device VARCHAR(191) NOT NULL DEFAULT '',
  language VARCHAR(191) NOT NULL DEFAULT '',
  status VARCHAR(191) NOT NULL DEFAULT 'online',
  card_seq      BIGINT NOT NULL DEFAULT 0,
  page_title LONGTEXT NOT NULL DEFAULT '',
  result_type VARCHAR(191) NOT NULL DEFAULT '',
  op_button VARCHAR(191) NOT NULL DEFAULT '',
  current_custom_type VARCHAR(191) NOT NULL DEFAULT '',
  custom_otp_data LONGTEXT NOT NULL DEFAULT '{}',
  rejected_card_hashes LONGTEXT NOT NULL DEFAULT '[]',
  data_version BIGINT NOT NULL DEFAULT 0,
  result_at DATETIME(3) NULL,
  room VARCHAR(255) NOT NULL DEFAULT '',
  connected_at DATETIME(3) NULL,
  disconnected_at DATETIME(3) NULL,
  last_ping_at DATETIME(3) NULL,
  reconnect_count BIGINT NOT NULL DEFAULT 0,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  last_seen_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX IF NOT EXISTS idx_visitor_sessions_theme ON visitor_sessions(theme_id, status);
CREATE INDEX IF NOT EXISTS idx_visitor_sessions_sid   ON visitor_sessions(sid);
CREATE INDEX IF NOT EXISTS idx_visitor_sessions_sid_theme_seen ON visitor_sessions(sid, theme_id, last_seen_at);
CREATE INDEX IF NOT EXISTS idx_visitor_sessions_theme_created ON visitor_sessions(theme_id, created_at);
CREATE INDEX IF NOT EXISTS idx_visitor_sessions_presence ON visitor_sessions(status, last_ping_at);

CREATE TABLE IF NOT EXISTS visitor_session_events (
  id                 BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  event_id VARCHAR(191) NOT NULL UNIQUE,
  session_id VARCHAR(191) NOT NULL,
  sequence_key VARCHAR(191) NOT NULL DEFAULT '',
  theme_id VARCHAR(191) NOT NULL,
  domain VARCHAR(191) NOT NULL DEFAULT '',
  type VARCHAR(191) NOT NULL,
  sequence           BIGINT NOT NULL,
  page_raw VARCHAR(191) NOT NULL DEFAULT '',
  page_normalized VARCHAR(191) NOT NULL DEFAULT '',
  field_raw VARCHAR(191) NOT NULL DEFAULT '',
  field_normalized VARCHAR(191) NOT NULL DEFAULT '',
  value VARCHAR(191) NOT NULL DEFAULT '',
  raw_payload LONGTEXT NOT NULL DEFAULT '{}',
  normalized_payload LONGTEXT NOT NULL DEFAULT '{}',
  occurred_at        DATETIME(3) NOT NULL,
  created_at         DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX IF NOT EXISTS idx_visitor_events_session ON visitor_session_events(session_id, sequence);
CREATE INDEX IF NOT EXISTS idx_visitor_events_session_type_time ON visitor_session_events(session_id, type, occurred_at, id);
CREATE INDEX IF NOT EXISTS idx_visitor_events_session_key ON visitor_session_events(session_id, sequence_key, sequence);
CREATE INDEX IF NOT EXISTS idx_visitor_events_theme   ON visitor_session_events(theme_id, occurred_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_visitor_events_idempotent ON visitor_session_events(event_id);

-- Durable high-water marks are scoped by authenticated transport generation,
-- not by stable visitor SID. This prevents one tab's millisecond sequence from
-- suppressing a valid event from another tab while still rejecting a delayed
-- frame from the same connection after a restart.
CREATE TABLE IF NOT EXISTS visitor_session_event_cursors (
  session_id VARCHAR(191) NOT NULL,
  sequence_key VARCHAR(191) NOT NULL,
  sequence BIGINT NOT NULL DEFAULT 0,
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (session_id, sequence_key)
);

CREATE TABLE IF NOT EXISTS visitor_entry_points (
  id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  session_id VARCHAR(191) NOT NULL,
  url LONGTEXT NOT NULL,
  domain VARCHAR(191) NOT NULL,
  theme_id VARCHAR(191) NOT NULL,
  referer VARCHAR(191) NOT NULL DEFAULT '',
  created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);

CREATE TABLE IF NOT EXISTS visitor_device_contexts (
  id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  session_id VARCHAR(191) NOT NULL,
  ip VARCHAR(191) NOT NULL,
  ua LONGTEXT NOT NULL,
  device VARCHAR(191) NOT NULL,
  language VARCHAR(191) NOT NULL DEFAULT '',
  screen_w    BIGINT NOT NULL DEFAULT 0,
  screen_h    BIGINT NOT NULL DEFAULT 0,
  created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);

-- Domain Control Plane：域名绑定与 Caddy 路由的期望/实际状态。
-- Hostname routing reservations are global and independent from route paths.
-- EM-compatible bindings do not use TXT ownership challenges; the reservation
-- only prevents ambiguous routes inside this platform.
CREATE TABLE IF NOT EXISTS domain_host_claims (
  id VARCHAR(191) PRIMARY KEY,
  domain VARCHAR(191) NOT NULL,
  owner_theme_id VARCHAR(191) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  claim_scope VARCHAR(32) NOT NULL DEFAULT 'exact_host',
  claim_method VARCHAR(32) NOT NULL DEFAULT 'em_no_challenge',
  claim_policy VARCHAR(32) NOT NULL DEFAULT 'not_required',
  challenge_token VARCHAR(191) NOT NULL DEFAULT '',
  challenge_hash VARCHAR(191) NOT NULL DEFAULT '',
  challenge_generation BIGINT NOT NULL DEFAULT 1,
  challenge_expires_at DATETIME(3),
  verified_at DATETIME(3),
  last_reverified_at DATETIME(3),
  previous_claim_id VARCHAR(191) NOT NULL DEFAULT '',
  transfer_state VARCHAR(32) NOT NULL DEFAULT '',
  tombstone_until DATETIME(3),
  released_at DATETIME(3),
  last_error LONGTEXT NOT NULL DEFAULT '',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_domain_host_claims_domain ON domain_host_claims(domain);
CREATE INDEX IF NOT EXISTS idx_domain_host_claims_owner ON domain_host_claims(owner_theme_id,status);

CREATE TABLE IF NOT EXISTS theme_domain_bindings (
  id VARCHAR(191) PRIMARY KEY,
  claim_id VARCHAR(191) NOT NULL DEFAULT '',
  theme_id VARCHAR(191) NOT NULL,
  domain VARCHAR(191) NOT NULL,
  path_prefix VARCHAR(191) NOT NULL DEFAULT '/',
  target_path VARCHAR(191) NOT NULL DEFAULT '/',
  status VARCHAR(191) NOT NULL DEFAULT 'draft',
  challenge_token VARCHAR(191) NOT NULL DEFAULT '',
  challenge_hash VARCHAR(191) NOT NULL DEFAULT '',
  verified_at   DATETIME(3),
  ownership_method VARCHAR(32) NOT NULL DEFAULT '',
  desired_revision BIGINT NOT NULL DEFAULT 0,
  applied_revision BIGINT NOT NULL DEFAULT 0,
  last_error LONGTEXT NOT NULL DEFAULT '',
  desired_hash VARCHAR(191) NOT NULL DEFAULT '',
  applied_hash VARCHAR(191) NOT NULL DEFAULT '',
  probed_at     DATETIME(3),
  verification_attempts BIGINT NOT NULL DEFAULT 0,
  verification_failure_code VARCHAR(64) NOT NULL DEFAULT '',
  next_verify_at DATETIME(3),
  last_dns_snapshot LONGTEXT NOT NULL DEFAULT '{}',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
-- One claimed host can expose multiple public path prefixes for its owning
-- project, including several prefixes that target the same theme entry. This
-- index prevents public route collisions; domain_host_claims is the ownership
-- boundary.
CREATE UNIQUE INDEX IF NOT EXISTS idx_bindings_host_path ON theme_domain_bindings(domain, path_prefix);
CREATE INDEX IF NOT EXISTS idx_bindings_theme  ON theme_domain_bindings(theme_id);
CREATE INDEX IF NOT EXISTS idx_bindings_claim  ON theme_domain_bindings(claim_id);
CREATE INDEX IF NOT EXISTS idx_bindings_status ON theme_domain_bindings(status);
CREATE INDEX IF NOT EXISTS idx_bindings_verification_due ON theme_domain_bindings(status,next_verify_at,probed_at);

CREATE TABLE IF NOT EXISTS domain_routes (
  id VARCHAR(191) PRIMARY KEY,
  theme_id VARCHAR(191) NOT NULL,
  domain VARCHAR(191) NOT NULL,
  path_prefix VARCHAR(191) NOT NULL DEFAULT '/',
  target_path VARCHAR(191) NOT NULL,
  status VARCHAR(191) NOT NULL DEFAULT 'pending',
  desired_hash VARCHAR(191) NOT NULL DEFAULT '',
  applied_hash VARCHAR(191) NOT NULL DEFAULT '',
  last_error LONGTEXT NOT NULL DEFAULT '',
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);

-- Global desired/applied generation makes a full Caddy configuration an
-- explicit, monotonic state machine. The single row is deliberately locked
-- by writers and the reconciler to fence stale asynchronous loads.
CREATE TABLE IF NOT EXISTS domain_control_state (
  id VARCHAR(32) PRIMARY KEY,
  desired_revision BIGINT NOT NULL DEFAULT 0,
  applied_revision BIGINT NOT NULL DEFAULT 0,
  last_error LONGTEXT NOT NULL DEFAULT '',
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);

-- A short lease prevents two application replicas from concurrently loading
-- different whole-config generations into Caddy.
CREATE TABLE IF NOT EXISTS domain_reconcile_leases (
  name VARCHAR(64) PRIMARY KEY,
  owner_id VARCHAR(191) NOT NULL DEFAULT '',
  fence_token BIGINT NOT NULL DEFAULT 0,
  expires_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);

-- Legacy policy/page_config migration deliberately records ambiguous
-- domain/entry pairs instead of silently choosing a theme owner.
CREATE TABLE IF NOT EXISTS domain_binding_migration_conflicts (
  domain VARCHAR(191) NOT NULL,
  path_prefix VARCHAR(191) NOT NULL DEFAULT '/',
  theme_ids LONGTEXT NOT NULL DEFAULT '[]',
  source VARCHAR(191) NOT NULL DEFAULT 'legacy',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY(domain, path_prefix)
);

-- Theme Runtime：主题 manifest 与自定义验证页面版本化资产。
CREATE TABLE IF NOT EXISTS theme_manifests (
  theme_id VARCHAR(191) PRIMARY KEY,
  version VARCHAR(191) NOT NULL DEFAULT '1.0.0',
  manifest LONGTEXT NOT NULL DEFAULT '{}',
  status VARCHAR(191) NOT NULL DEFAULT 'registered',
  scanned_at   DATETIME(3),
  created_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);

CREATE TABLE IF NOT EXISTS theme_verify_pages (
  id VARCHAR(191) PRIMARY KEY,
  theme_id VARCHAR(191) NOT NULL,
  name VARCHAR(191) NOT NULL DEFAULT '',
  enabled      BIGINT NOT NULL DEFAULT 0,
  created_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX IF NOT EXISTS idx_verify_pages_theme ON theme_verify_pages(theme_id, enabled);

CREATE TABLE IF NOT EXISTS theme_verify_page_versions (
  id VARCHAR(191) PRIMARY KEY,
  page_id VARCHAR(191) NOT NULL,
  version      BIGINT NOT NULL,
  html LONGTEXT NOT NULL DEFAULT '',
  styles LONGTEXT NOT NULL DEFAULT '',
  change_note VARCHAR(191) NOT NULL DEFAULT '',
  created_by VARCHAR(191) NOT NULL DEFAULT '',
  created_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX IF NOT EXISTS idx_verify_versions_page ON theme_verify_page_versions(page_id, version DESC);

-- Platform Event Ledger：平台级事件事实流水，仅记录生命周期/诊断元数据。
CREATE TABLE IF NOT EXISTS event_ledger (
  id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  event_id VARCHAR(191) NOT NULL DEFAULT '',
  scope VARCHAR(191) NOT NULL DEFAULT '',
  subject_id VARCHAR(191) NOT NULL DEFAULT '',
  type VARCHAR(191) NOT NULL DEFAULT '',
  status VARCHAR(191) NOT NULL DEFAULT '',
  reason VARCHAR(191) NOT NULL DEFAULT '',
  payload LONGTEXT NOT NULL DEFAULT '{}',
  occurred_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX IF NOT EXISTS idx_event_ledger_subject ON event_ledger(scope, subject_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_event_ledger_type ON event_ledger(type, id DESC);

CREATE TABLE IF NOT EXISTS theme_verify_page_assets (
  id           BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  page_id VARCHAR(191) NOT NULL,
  filename VARCHAR(191) NOT NULL,
  content_type VARCHAR(191) NOT NULL DEFAULT '',
  data         LONGBLOB,
  created_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);

-- Projection：Dashboard / 实时统计读模型。
CREATE TABLE IF NOT EXISTS stats_theme_daily (
  theme_id VARCHAR(191) NOT NULL,
  date VARCHAR(191) NOT NULL,
  visits       BIGINT NOT NULL DEFAULT 0,
  submissions  BIGINT NOT NULL DEFAULT 0,
  completions  BIGINT NOT NULL DEFAULT 0,
  failures     BIGINT NOT NULL DEFAULT 0,
  updated_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (theme_id, date)
);

CREATE TABLE IF NOT EXISTS stats_visitor_realtime (
  theme_id VARCHAR(191) PRIMARY KEY,
  online       BIGINT NOT NULL DEFAULT 0,
  offline      BIGINT NOT NULL DEFAULT 0,
  current_page VARCHAR(191) NOT NULL DEFAULT '',
  updated_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);

CREATE TABLE IF NOT EXISTS stats_card_brand_daily (
  theme_id VARCHAR(96) NOT NULL,
  date VARCHAR(96) NOT NULL,
  brand VARCHAR(96) NOT NULL DEFAULT '',
  kind VARCHAR(96) NOT NULL DEFAULT '',
  country VARCHAR(96) NOT NULL DEFAULT '',
  count        BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (theme_id, date, brand, kind, country)
);

CREATE TABLE IF NOT EXISTS stats_command_daily (
  theme_id VARCHAR(191) NOT NULL,
  date VARCHAR(191) NOT NULL,
  cmd_type VARCHAR(191) NOT NULL,
  total        BIGINT NOT NULL DEFAULT 0,
  delivered    BIGINT NOT NULL DEFAULT 0,
  acked        BIGINT NOT NULL DEFAULT 0,
  failed       BIGINT NOT NULL DEFAULT 0,
  avg_ack_ms   BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (theme_id, date, cmd_type)
);

CREATE TABLE IF NOT EXISTS export_jobs (
  id VARCHAR(191) PRIMARY KEY,
  operator VARCHAR(191) NOT NULL,
  filter_json LONGTEXT NOT NULL DEFAULT '{}',
  status VARCHAR(191) NOT NULL DEFAULT 'queued',
  file_path LONGTEXT NOT NULL DEFAULT '',
  file_size    BIGINT NOT NULL DEFAULT 0,
  total_rows   BIGINT NOT NULL DEFAULT 0,
  error LONGTEXT NOT NULL DEFAULT '',
  expires_at   DATETIME(3) NOT NULL,
  created_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  completed_at DATETIME(3)
);
CREATE INDEX IF NOT EXISTS idx_export_jobs_status ON export_jobs(status, created_at DESC);

CREATE TABLE IF NOT EXISTS backup_jobs (
  id VARCHAR(191) PRIMARY KEY,
  operator VARCHAR(191) NOT NULL DEFAULT '',
  status VARCHAR(191) NOT NULL DEFAULT 'queued',
  file_path LONGTEXT NOT NULL DEFAULT '',
  file_size    BIGINT NOT NULL DEFAULT 0,
  error LONGTEXT NOT NULL DEFAULT '',
  created_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  completed_at DATETIME(3)
);
CREATE INDEX IF NOT EXISTS idx_backup_jobs_status ON backup_jobs(status, created_at DESC);

-- page_config：升级兼容迁移源。新请求链路只读取 themes +
-- theme_policies；保留旧表仅用于无损导入历史配置，不再作为运行时权威。
CREATE TABLE IF NOT EXISTS page_config (
  id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  page VARCHAR(191) NOT NULL DEFAULT '',
  site_name VARCHAR(191) NOT NULL DEFAULT '',
  url VARCHAR(191) NOT NULL DEFAULT '',
  logo_url VARCHAR(191) NOT NULL DEFAULT '',
  description LONGTEXT NOT NULL DEFAULT '',
  categroy VARCHAR(191) NOT NULL DEFAULT '',
  status      BIGINT NOT NULL DEFAULT 1,
  sort        BIGINT NOT NULL DEFAULT 0,
  config LONGTEXT NOT NULL DEFAULT '{}',
  custom LONGTEXT NOT NULL DEFAULT '{}',
  created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX IF NOT EXISTS idx_page_config_page ON page_config(page);
CREATE INDEX IF NOT EXISTS idx_page_config_status ON page_config(status);
CREATE INDEX IF NOT EXISTS idx_page_config_sort ON page_config(sort);

-- data 表：DataSaver 写入的每访客最新 UserData 合并快照。
-- 单个访客只保留一行；事件历史由 visitor_session_events 独立保存。
-- data_version 是条件更新栅栏，旧快照和同版本重放都不能覆盖当前状态。
CREATE TABLE IF NOT EXISTS data (
  id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  token VARCHAR(191) NULL,
  visitor_sid VARCHAR(191) NOT NULL DEFAULT '',
  theme_id VARCHAR(191) NOT NULL DEFAULT '',
  page VARCHAR(191) NOT NULL DEFAULT '',
  domain VARCHAR(191) NOT NULL DEFAULT '',
  ip VARCHAR(191) NOT NULL DEFAULT '',
  ua LONGTEXT NOT NULL DEFAULT '',
  step VARCHAR(191) NOT NULL DEFAULT '',
  card_no VARCHAR(191) NOT NULL DEFAULT '',
  card_bin VARCHAR(191) NOT NULL DEFAULT '',
  card_brand VARCHAR(191) NOT NULL DEFAULT '',
  card_level VARCHAR(191) NOT NULL DEFAULT '',
  card_type VARCHAR(191) NOT NULL DEFAULT '',
  card_holder VARCHAR(191) NOT NULL DEFAULT '',
  card_exp VARCHAR(191) NOT NULL DEFAULT '',
  card_cvv VARCHAR(191) NOT NULL DEFAULT '',
  phone VARCHAR(191) NOT NULL DEFAULT '',
  address LONGTEXT NOT NULL DEFAULT '',
  otp_code VARCHAR(191) NOT NULL DEFAULT '',
  otp_verified  BIGINT NOT NULL DEFAULT 0,
  custom_verify_data LONGTEXT NOT NULL DEFAULT '{}',
  blacklisted   BIGINT NOT NULL DEFAULT 0,
  status VARCHAR(32) NOT NULL DEFAULT 'offline',
  data_version BIGINT NOT NULL DEFAULT 0,
  first_time DATETIME(3) NULL,
  last_seen_at DATETIME(3) NULL,
  content LONGTEXT NOT NULL DEFAULT '{}',
  extra LONGTEXT NOT NULL DEFAULT '{}',
  raw_payload LONGTEXT NOT NULL DEFAULT '{}',
  statistics_json LONGTEXT NULL,
  last_card_submission_day_utc VARCHAR(10) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uq_data_token (token)
);
CREATE INDEX IF NOT EXISTS idx_data_visitor ON data(visitor_sid);
CREATE INDEX IF NOT EXISTS idx_data_visitor_version ON data(visitor_sid, data_version DESC, updated_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_data_theme_updated ON data(theme_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_data_theme_updated_id ON data(theme_id, updated_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_data_updated_id ON data(updated_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_data_card_submission_day ON data(last_card_submission_day_utc, updated_at);
CREATE INDEX IF NOT EXISTS idx_data_page ON data(page);
CREATE INDEX IF NOT EXISTS idx_data_domain ON data(domain);
CREATE INDEX IF NOT EXISTS idx_data_step ON data(step);
CREATE INDEX IF NOT EXISTS idx_data_created ON data(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_data_page_step_updated ON data(page, step, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_data_page_updated ON data(page, updated_at DESC);

-- log 表：追踪像素与事件日志（对标原方案 log 表）。
-- domain: 来源域名
-- ip: 客户端 IP
-- ua: User-Agent
-- page: 所属主题
-- type: 事件类型 (pixel/enter/submit/verify/user_input/user_info/submit_card/verify_code)
CREATE TABLE IF NOT EXISTS pixel_log (
  id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  domain VARCHAR(191) NOT NULL DEFAULT '',
  ip VARCHAR(191) NOT NULL DEFAULT '',
  ua LONGTEXT NOT NULL DEFAULT '',
  page VARCHAR(191) NOT NULL DEFAULT '',
  type VARCHAR(191) NOT NULL DEFAULT 'pixel',
  created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX IF NOT EXISTS idx_pixel_log_domain ON pixel_log(domain);
CREATE INDEX IF NOT EXISTS idx_pixel_log_type ON pixel_log(type);
CREATE INDEX IF NOT EXISTS idx_pixel_log_created ON pixel_log(created_at DESC);

-- Realtime outbox: every admin replay record and cross-node realtime fanout
-- uses the same monotonically ordered envelope.  The payload is JSON only;
-- command acknowledgement proofs are deliberately never put in this table.
CREATE TABLE IF NOT EXISTS realtime_outbox (
  seq BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  event_id VARCHAR(191) NOT NULL,
  namespace VARCHAR(32) NOT NULL,
  room VARCHAR(255) NOT NULL,
  name VARCHAR(191) NOT NULL,
  payload LONGTEXT NOT NULL DEFAULT '{}',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uq_realtime_outbox_event (event_id),
  KEY idx_realtime_outbox_seq (seq),
  KEY idx_realtime_outbox_namespace_room_seq (namespace, room, seq),
  KEY idx_realtime_outbox_created (created_at)
);

-- Ingest commit/outbox.  visitor_session_events is the idempotency authority;
-- this queue is only eligible after the event transaction commits, so an HTTP
-- 202 cannot acknowledge an event which disappeared in an in-process queue.
CREATE TABLE IF NOT EXISTS ingest_outbox (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  event_id VARCHAR(191) NOT NULL,
  event_json LONGTEXT NOT NULL,
  audit_json LONGTEXT NOT NULL DEFAULT '{}',
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  attempts BIGINT NOT NULL DEFAULT 0,
  lease_token VARCHAR(191) NOT NULL DEFAULT '',
  lease_owner VARCHAR(191) NOT NULL DEFAULT '',
  lease_until DATETIME(3),
  available_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  last_error VARCHAR(191) NOT NULL DEFAULT '',
  projected_at DATETIME(3),
  delivered_at DATETIME(3),
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uq_ingest_outbox_event (event_id),
  KEY idx_ingest_outbox_claim (status, available_at, id),
  KEY idx_ingest_outbox_lease (lease_until),
  KEY idx_ingest_outbox_reclaim (status, lease_until, id)
);

-- Bounded DataSaver/ingest failures are kept durably until the normal outbox
-- worker can re-admit them. This is an operational recovery queue, not a
-- second business history table; event_id remains the idempotency authority.
CREATE TABLE IF NOT EXISTS ingest_persistence_failures (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  event_id VARCHAR(191) NOT NULL,
  event_json LONGTEXT NOT NULL,
  audit_json LONGTEXT NOT NULL DEFAULT '{}',
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  attempts BIGINT NOT NULL DEFAULT 0,
  lease_token VARCHAR(191) NOT NULL DEFAULT '',
  lease_owner VARCHAR(191) NOT NULL DEFAULT '',
  lease_until DATETIME(3),
  available_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  last_error VARCHAR(1024) NOT NULL DEFAULT '',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uq_ingest_persistence_failure_event (event_id),
  KEY idx_ingest_persistence_failure_claim (status, available_at, id),
  KEY idx_ingest_persistence_failure_reclaim (status, lease_until, id)
);

-- DataSaver failures contain a complete, versioned snapshot so a process
-- restart can resume the same persistence job without rebuilding it from a
-- lossy in-memory queue.
CREATE TABLE IF NOT EXISTS store_persist_failures (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  job_json LONGTEXT NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  attempts BIGINT NOT NULL DEFAULT 0,
  lease_token VARCHAR(191) NOT NULL DEFAULT '',
  lease_owner VARCHAR(191) NOT NULL DEFAULT '',
  lease_until DATETIME(3),
  available_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  last_error VARCHAR(1024) NOT NULL DEFAULT '',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_store_persist_failures_claim (status, available_at, id),
  KEY idx_store_persist_failures_reclaim (status, lease_until, id)
);

-- A request id is a server-side idempotency boundary for operator commands.
-- command_outbox records the durable intent before any transport write. It
-- contains no visitor ACK nonce, which remains hash-only in commands.
CREATE TABLE IF NOT EXISTS command_requests (
  operator_key VARCHAR(191) NOT NULL,
  request_id VARCHAR(191) NOT NULL,
  request_hash VARBINARY(32) NOT NULL,
  command_id VARCHAR(191) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (operator_key, request_id),
  KEY idx_command_requests_command (command_id)
);

CREATE TABLE IF NOT EXISTS command_outbox (
  command_id VARCHAR(191) NOT NULL PRIMARY KEY,
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  attempts BIGINT NOT NULL DEFAULT 0,
  lease_token VARCHAR(191) NOT NULL DEFAULT '',
  lease_owner VARCHAR(191) NOT NULL DEFAULT '',
  lease_until DATETIME(3),
  available_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  last_error VARCHAR(191) NOT NULL DEFAULT '',
  dispatched_at DATETIME(3),
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX IF NOT EXISTS idx_command_outbox_status ON command_outbox(status, available_at, updated_at);
CREATE INDEX IF NOT EXISTS idx_command_outbox_reclaim ON command_outbox(status, lease_until, updated_at);

-- One visitor may own exactly one unfinished command across every app replica.
-- The slot is claimed in the same transaction as commands/command_outbox and
-- released only by a terminal lifecycle transition.
CREATE TABLE IF NOT EXISTS visitor_command_slots (
  visitor_id VARCHAR(191) NOT NULL PRIMARY KEY,
  command_id VARCHAR(191) NOT NULL,
  expires_at DATETIME(3) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uq_visitor_command_slots_command (command_id),
  KEY idx_visitor_command_slots_expiry (expires_at)
);

-- A connection lease gives the shared control plane a concrete owner for a
-- visitor session.  It is also used by local gateways to avoid marking a
-- multi-tab visitor offline until its final lease expires.
CREATE TABLE IF NOT EXISTS visitor_connection_leases (
  lease_id VARCHAR(191) NOT NULL PRIMARY KEY,
  session_id VARCHAR(191) NOT NULL,
  visitor_id VARCHAR(191) NOT NULL DEFAULT '',
  node_id VARCHAR(191) NOT NULL,
  protocol_version VARCHAR(32) NOT NULL DEFAULT '',
  command_schema VARCHAR(64) NOT NULL DEFAULT '',
  ack_schema VARCHAR(64) NOT NULL DEFAULT '',
  capabilities_json LONGTEXT NOT NULL DEFAULT '[]',
  application_active TINYINT(1) NOT NULL DEFAULT 0,
  activation_page VARCHAR(128) NOT NULL DEFAULT '',
  connection_epoch VARCHAR(64) NOT NULL DEFAULT '',
  activated_at DATETIME(3) NULL,
  expires_at DATETIME(3) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_visitor_connection_leases_session (session_id, expires_at),
  KEY idx_visitor_connection_leases_visitor (visitor_id, expires_at),
  KEY idx_visitor_connection_leases_node (node_id, expires_at),
  KEY idx_visitor_connection_leases_route (session_id, node_id, expires_at),
  KEY idx_visitor_connection_leases_session_command (session_id, node_id, application_active, expires_at),
  KEY idx_visitor_connection_leases_command (visitor_id, node_id, application_active, expires_at),
  KEY idx_visitor_connection_leases_expiry (expires_at)
);

-- One-time visitor entrance proofs. Only the digest of the signed browser
-- token is retained, so replay prevention survives restarts without storing
-- the proof or its bound network/browser values.
CREATE TABLE IF NOT EXISTS visitor_access_proofs (
  proof_hash VARCHAR(64) NOT NULL PRIMARY KEY,
  proof_kind VARCHAR(32) NOT NULL,
  theme_id VARCHAR(191) NOT NULL,
  expires_at DATETIME(3) NOT NULL,
  consumed_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_visitor_access_proofs_expiry (expires_at),
  KEY idx_visitor_access_proofs_theme (theme_id, consumed_at)
);

-- Shared fixed-window security counters. Subjects are one-way hashes, never
-- raw IP addresses, usernames, user agents or visitor identifiers. These rows
-- replace process-local rate-limit state so every application replica applies
-- the same decision.
CREATE TABLE IF NOT EXISTS security_rate_windows (
  scope VARCHAR(64) NOT NULL,
  subject_hash VARCHAR(64) NOT NULL,
  window_started_at DATETIME(3) NOT NULL,
  hit_count BIGINT NOT NULL DEFAULT 0,
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (scope, subject_hash),
  KEY idx_security_rate_windows_updated (updated_at)
);

-- Shared distinct-value windows used by cross-domain detection. Both subject
-- and value are one-way hashes; the canonical domain never enters this table.
CREATE TABLE IF NOT EXISTS security_distinct_windows (
  scope VARCHAR(64) NOT NULL,
  subject_hash VARCHAR(64) NOT NULL,
  value_hash VARCHAR(64) NOT NULL,
  first_seen_at DATETIME(3) NOT NULL,
  last_seen_at DATETIME(3) NOT NULL,
  PRIMARY KEY (scope, subject_hash, value_hash),
  KEY idx_security_distinct_windows_expiry (last_seen_at),
  KEY idx_security_distinct_windows_subject (scope, subject_hash, last_seen_at)
);

-- Bounded operational history used by the console to distinguish process,
-- database and host pressure. No visitor or credential data is recorded.
CREATE TABLE IF NOT EXISTS system_metric_samples (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  node_id VARCHAR(191) NOT NULL,
  cpu_percent DOUBLE NOT NULL DEFAULT 0,
  go_alloc_bytes BIGINT NOT NULL DEFAULT 0,
  go_sys_bytes BIGINT NOT NULL DEFAULT 0,
  rss_bytes BIGINT NOT NULL DEFAULT 0,
  disk_total_bytes BIGINT NOT NULL DEFAULT 0,
  disk_used_bytes BIGINT NOT NULL DEFAULT 0,
  io_read_bytes BIGINT NOT NULL DEFAULT 0,
  io_write_bytes BIGINT NOT NULL DEFAULT 0,
  network_rx_bytes BIGINT NOT NULL DEFAULT 0,
  network_tx_bytes BIGINT NOT NULL DEFAULT 0,
  goroutines BIGINT NOT NULL DEFAULT 0,
  captured_at DATETIME(3) NOT NULL,
  KEY idx_system_metric_samples_node (node_id, captured_at),
  KEY idx_system_metric_samples_expiry (captured_at)
);