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
