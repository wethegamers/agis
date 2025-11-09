FROM golang:1.21-alpine as builder

# Build arguments for version information
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build with version information injected
RUN go build -ldflags="-X agis-bot/internal/version.Version=${VERSION} -X agis-bot/internal/version.GitCommit=${GIT_COMMIT} -X agis-bot/internal/version.BuildDate=${BUILD_DATE}" -o agis-bot .

FROM alpine:3.18
WORKDIR /app

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/agis-bot ./agis-bot
COPY --from=builder /app/.env.example ./.env.example

# Expose both Discord and HTTP server ports
EXPOSE 9090

ENTRYPOINT ["./agis-bot"]
