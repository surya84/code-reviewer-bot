FROM golang:1.24.4-alpine

WORKDIR /app

COPY . .

RUN go mod download
RUN go build -o /usr/local/bin/code-reviewer-bot .

ENTRYPOINT ["code-reviewer-bot"]
