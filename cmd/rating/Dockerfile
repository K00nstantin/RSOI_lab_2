FROM golang:1.21-alpine

WORKDIR /app

# Копируем весь проект
COPY . .

RUN go mod download
RUN go build -o rating cmd/rating/main.go

EXPOSE 8050

CMD ["./rating"]