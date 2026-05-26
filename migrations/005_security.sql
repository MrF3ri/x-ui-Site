-- 005_security.sql

-- Audit log for all sensitive actions
CREATE TABLE IF NOT EXISTS audit_logs (
  id          BIGSERIAL PRIMARY KEY,
  vendor_id   BIGINT NULL,
  user_id     BIGINT NULL,
  actor_role  TEXT   NOT NULL DEFAULT '',
  action      TEXT   NOT NULL,
  resource    TEXT   NOT NULL DEFAULT '',
  resource_id BIGINT NULL,
  ip          TEXT   NOT NULL DEFAULT '',
  user_agent  TEXT   NOT NULL DEFAULT '',
  status      TEXT   NOT NULL DEFAULT 'ok',  -- ok | denied | error
  detail      TEXT   NOT NULL DEFAULT '',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_audit_logs_vendor   ON audit_logs(vendor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user     ON audit_logs(user_id,   created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action   ON audit_logs(action,    created_at DESC);

-- Rate-limit counters (simple sliding window via postgres)
CREATE TABLE IF NOT EXISTS rate_limit_buckets (
  id         BIGSERIAL PRIMARY KEY,
  key        TEXT        NOT NULL,
  count      INT         NOT NULL DEFAULT 1,
  window_end TIMESTAMPTZ NOT NULL,
  UNIQUE(key, window_end)
);
CREATE INDEX IF NOT EXISTS idx_rate_limit_key ON rate_limit_buckets(key, window_end);
