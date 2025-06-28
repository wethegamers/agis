FROM golang:1.21-alpine as builder
WORKDIR /app
COPY . .
RUN go build -o agis-bot .

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/agis-bot ./agis-bot
COPY --from=builder /app/.env.example ./.env.example
EXPOSE 8080
ENTRYPOINT ["./agis-bot"]
