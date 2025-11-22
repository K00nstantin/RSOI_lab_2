FROM golang:1.21-alpine

WORKDIR /app

# Копируем весь проект
COPY . .

RUN go mod download
RUN go build -o gateway cmd/gateway/main.go

EXPOSE 8080

CMD ["./gateway"]