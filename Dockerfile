FROM golang:1.24

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build binary for the platform Docker is building for
RUN go build -o reviewer ./cmd/reviewer 

COPY config/ ./config/

ENTRYPOINT ["/app/reviewer"]