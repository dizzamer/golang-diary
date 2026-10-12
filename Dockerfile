FROM golang:1.23-alpine AS builder

WORKDIR /app

# Копируем созданные файлы модулей
COPY go.mod ./
# Если команда go mod tidy создала go.sum, раскомментируйте строку ниже:
# COPY go.sum ./ 
RUN go mod download

# Копируем всё остальное (main.go, index.html)
COPY . .

# Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/web-app main.go

# --- Финальный легковесный образ ---
FROM alpine:latest
WORKDIR /root/

# Копируем скомпилированное приложение И ваш HTML-шаблон
COPY --from=builder /app/web-app .
COPY --from=builder /app/index.html .

EXPOSE 8080

CMD ["./web-app"]
