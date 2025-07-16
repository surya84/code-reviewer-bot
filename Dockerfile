
FROM golang:1.24.4-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

COPY config/ ./config/

# Build the Go binary
RUN go build -o /usr/local/bin/code-reviewer-bot .

# Run the binary by default
ENTRYPOINT ["code-reviewer-bot"]
