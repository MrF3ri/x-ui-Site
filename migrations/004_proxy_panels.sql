-- 004_proxy_panels.sql

-- xui panel registry per vendor
CREATE TABLE IF NOT EXISTS xui_panels (
  id           BIGSERIAL PRIMARY KEY,
  vendor_id    BIGINT NOT NULL REFERENCES vendors(id),
  name         TEXT   NOT NULL,
  url          TEXT   NOT NULL,
  token        TEXT   NOT NULL,
  inbound_id   BIGINT NOT NULL,
  is_active    BOOLEAN NOT NULL DEFAULT TRUE,
  health       TEXT    NOT NULL DEFAULT 'unknown',
  last_checked TIMESTAMPTZ NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at   TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS idx_xui_panels_vendor ON xui_panels(vendor_id) WHERE deleted_at IS NULL;

-- extend proxy_services with uuid and config fields
ALTER TABLE proxy_services ADD COLUMN IF NOT EXISTS uuid TEXT NOT NULL DEFAULT '';
ALTER TABLE proxy_services ADD COLUMN IF NOT EXISTS protocol TEXT NOT NULL DEFAULT '';
ALTER TABLE proxy_services ADD COLUMN IF NOT EXISTS traffic_used_gb  INT NOT NULL DEFAULT 0;
ALTER TABLE proxy_services ADD COLUMN IF NOT EXISTS traffic_limit_gb INT NOT NULL DEFAULT 0;
ALTER TABLE proxy_services ADD COLUMN IF NOT EXISTS duration_days    INT NOT NULL DEFAULT 30;

CREATE UNIQUE INDEX IF NOT EXISTS idx_proxy_services_uuid ON proxy_services(uuid) WHERE deleted_at IS NULL;
