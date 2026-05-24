FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install git so `go mod download` can resolve VCS deps
RUN apk add --no-cache git ca-certificates

# Cache module downloads as a separate layer
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 go build -o garudapanel ./cmd/server

# ── Runtime image ──────────────────────────────────────────────────────────
FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/garudapanel   /app/garudapanel
COPY --from=builder /app/templates     /app/templates
COPY --from=builder /app/public        /app/public
COPY --from=builder /app/migrations    /app/migrations

EXPOSE 8080
CMD ["/app/garudapanel"]