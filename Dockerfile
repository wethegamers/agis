FROM golang:1.24-alpine as builder

# Build arguments for version information
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown
ARG GITHUB_TOKEN

WORKDIR /app

# Configure git for private repo access
RUN apk add --no-cache git
RUN git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"

COPY go.mod go.sum ./

# Remove replace directive for CI builds (use git fetch)
RUN sed -i '/^replace/d' go.mod

RUN go mod download

COPY . .

# Build with version information injected
RUN go build -ldflags="-X github.com/wethegamers/agis-core/version.Version=${VERSION} -X github.com/wethegamers/agis-core/version.GitCommit=${GIT_COMMIT} -X github.com/wethegamers/agis-core/version.BuildDate=${BUILD_DATE}" -o agis-bot .

FROM alpine:3.18
WORKDIR /app

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/agis-bot ./agis-bot
COPY --from=builder /app/.env.example ./.env.example

# Expose both Discord and HTTP server ports
EXPOSE 9090

ENTRYPOINT ["./agis-bot"]
