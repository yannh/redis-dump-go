FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o upstash-redis-dump .

FROM alpine:latest

RUN apk --no-cache add ca-certificates redis

RUN addgroup -g 1001 -S appuser && \
    adduser -u 1001 -S appuser -G appuser

WORKDIR /app
COPY --from=builder /app/upstash-redis-dump .
RUN chown appuser:appuser upstash-redis-dump
USER appuser

ENTRYPOINT ["./upstash-redis-dump"]
CMD ["-h"]
