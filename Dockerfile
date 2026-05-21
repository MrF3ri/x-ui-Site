FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY . .
RUN go build -o garudapanel ./cmd/server

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/garudapanel /app/garudapanel
COPY --from=builder /app/templates /app/templates
COPY --from=builder /app/public /app/public
COPY --from=builder /app/migrations /app/migrations
EXPOSE 8080
CMD ["/app/garudapanel"]
