FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server main.go

FROM alpine:latest

WORKDIR /app

RUN addgroup -S app && adduser -S app -G app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/server /app/server

# Copy SQL migrations so the app can apply them on container startup.
COPY --from=builder /app/migrations /app/migrations

RUN chown -R app:app /app

USER app

ENV APP_PORT=8080
ENV MIGRATIONS_DIR=/app/migrations

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/docs/index.html > /dev/null 2>&1 || exit 1

CMD ["/app/server"]
