FROM golang:1.21-alpine AS builder

WORKDIR /app

# Копируем файлы зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o library cmd/library/main.go

# Финальный образ
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Копируем бинарник
COPY --from=builder /app/library .

# Создаем non-root пользователя для безопасности
RUN adduser -D -s /bin/sh appuser
USER appuser

# Healthcheck (использует ваш /manage/health endpoint)
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8060/manage/health || exit 1

EXPOSE 8060

CMD ["./library"]