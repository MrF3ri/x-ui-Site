CREATE TABLE IF NOT EXISTS catalog_items (
  id BIGSERIAL PRIMARY KEY,
  vendor_id BIGINT NOT NULL REFERENCES vendors(id),
  slug TEXT NOT NULL,
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  protocol TEXT NOT NULL,
  inbound_id BIGINT NOT NULL,
  xui_node_id BIGINT NOT NULL,
  traffic_limit_gb INT NOT NULL,
  duration_days INT NOT NULL,
  price_toman BIGINT NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  auto_provision BOOLEAN NOT NULL DEFAULT TRUE,
  renewal_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  country_code TEXT NOT NULL DEFAULT 'IR',
  stock_status TEXT NOT NULL DEFAULT 'in_stock',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ NULL,
  UNIQUE(vendor_id, slug)
);
CREATE INDEX IF NOT EXISTS idx_catalog_items_vendor_id ON catalog_items(vendor_id);
CREATE INDEX IF NOT EXISTS idx_catalog_items_vendor_slug ON catalog_items(vendor_id, slug);
CREATE INDEX IF NOT EXISTS idx_catalog_items_active ON catalog_items(vendor_id, is_active) WHERE deleted_at IS NULL;
