# ── Build ──
FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o jmg .

# ── Runtime ──
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -h /app jmg
USER jmg
WORKDIR /app
COPY --from=builder /build/jmg .
RUN mkdir -p /app/data

LABEL org.opencontainers.image.source=https://github.com/woor1668/jmg

EXPOSE 8080
ENTRYPOINT ["./jmg"]
CMD ["--config", "/app/config.yaml"]
