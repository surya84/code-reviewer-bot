# Build stage
FROM golang:1.24.4-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags '-w -s' -o code-reviewer-bot .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/code-reviewer-bot .

COPY --from=builder /app/config/ ./config/

RUN chmod +x ./code-reviewer-bot

# Run the application
ENTRYPOINT ["/app/code-reviewer-bot"]