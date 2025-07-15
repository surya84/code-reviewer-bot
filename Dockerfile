FROM golang:1.24

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build binary for the platform Docker is building for
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/reviewer

COPY config/ ./config/

CMD ["./server"]