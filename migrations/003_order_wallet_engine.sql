CREATE TABLE IF NOT EXISTS order_idempotency_keys (
  id BIGSERIAL PRIMARY KEY,
  vendor_id BIGINT NOT NULL REFERENCES vendors(id),
  user_id BIGINT NOT NULL REFERENCES users(id),
  idempotency_key TEXT NOT NULL,
  order_id BIGINT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(vendor_id, user_id, idempotency_key)
);

ALTER TABLE orders ADD COLUMN IF NOT EXISTS lifecycle_state TEXT NOT NULL DEFAULT 'pending';
ALTER TABLE orders ADD COLUMN IF NOT EXISTS idempotency_key TEXT NULL;

CREATE TABLE IF NOT EXISTS provisioning_jobs (
  id BIGSERIAL PRIMARY KEY,
  vendor_id BIGINT NOT NULL REFERENCES vendors(id),
  order_id BIGINT NOT NULL REFERENCES orders(id),
  status TEXT NOT NULL DEFAULT 'pending',
  retries INT NOT NULL DEFAULT 0,
  last_error TEXT NULL,
  dead_letter BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ NULL
);

CREATE TABLE IF NOT EXISTS proxy_services (
  id BIGSERIAL PRIMARY KEY,
  vendor_id BIGINT NOT NULL REFERENCES vendors(id),
  user_id BIGINT NOT NULL REFERENCES users(id),
  order_id BIGINT NULL REFERENCES orders(id),
  panel_id BIGINT NULL,
  subscription_url TEXT,
  qr_payload TEXT,
  config_payload TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  expires_at TIMESTAMPTZ NULL,
  traffic_used_bytes BIGINT NOT NULL DEFAULT 0,
  traffic_limit_bytes BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ NULL
);
