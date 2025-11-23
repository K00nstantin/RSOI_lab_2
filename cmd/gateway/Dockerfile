FROM golang:1.21-alpine AS builder

WORKDIR /app

# Сначала копируем только go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Затем копируем весь исходный код
COPY . .

# Собираем приложение (учитываем структуру папки RSOI_lab_2)
RUN go build -o gateway ./cmd/gateway/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates wget

WORKDIR /root/
COPY --from=builder /app/gateway .
COPY wait-for.sh .

EXPOSE 8080
CMD ["./gateway"]