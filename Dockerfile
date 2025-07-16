# --- Build Stage ---
FROM golang:1.24.4-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

# Build the CLI binary
RUN go build -o code-reviewer-bot ./cmd/reviewer

# --- Runtime Stage ---
FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy the binary and config folder
COPY --from=builder /app/code-reviewer-bot /usr/local/bin/code-reviewer-bot
COPY --from=builder /app/config /app/config

ENTRYPOINT ["code-reviewer-bot"]