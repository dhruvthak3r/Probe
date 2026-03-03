FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o probe cmd/probe/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/probe .
COPY --from=builder /app/migrations ./migrations
EXPOSE 8080
CMD ["./probe"]