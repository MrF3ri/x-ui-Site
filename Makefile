.PHONY: dev logs rebuild migrate seed test

dev:
	docker compose up -d --build

logs:
	docker compose logs -f app postgres caddy

rebuild:
	docker compose down
	docker compose up -d --build

migrate:
	docker compose run --rm app /app/garudapanel

seed:
	APP_ENV=development docker compose run --rm app /app/garudapanel

test:
	go test ./...
