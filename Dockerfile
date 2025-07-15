# Start from official Golang image
FROM golang:1.24.4-alpine

# Set working directory inside container
WORKDIR /app

# Copy all files
COPY . .

# Build the Go binary
RUN go build -o reviewer ./cmd/reviewer

# Set entrypoint
ENTRYPOINT ["/app/reviewer"]
