# GarudaPanel

## End-to-end verification

```bash
docker compose up -d --build
curl -s http://127.0.0.1:8080/health
curl -i http://127.0.0.1:8080/store/demo
curl -i http://127.0.0.1:8080/store/demo/products
curl -i http://127.0.0.1:8080/store/demo/products/premium-vless
```

Expected:
- `/health` returns JSON with app/postgres/redis/minio statuses.
- storefront URLs return HTML pages without panic.

Additional dev routes:

- Dashboard (requires login JWT): `/dashboard`
- Dashboard services list: `/dashboard/services`
- Service detail (with QR + renew): `/dashboard/services/:id`
- Wallet: `/dashboard/wallet`
- Orders: `/dashboard/orders`

The renewal API endpoint (authenticated user) is:

```
POST /api/v1/order/renew/:serviceID
```

It performs wallet deduction, creates an order, enqueues provisioning, and extends service expiry transactionally.
