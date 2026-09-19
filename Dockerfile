FROM golang:1.26-alpine3.23 AS builder

WORKDIR /src

# Download dependencies in a separate layer to reuse Docker's build cache.
COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /out/diary-service ./cmd/diary-service

FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S -G app app

WORKDIR /app

COPY --from=builder /out/diary-service ./diary-service

# The application runs migrations from the relative file://migrations path.
COPY --from=builder /src/migrations ./migrations

USER app

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -q -O /dev/null "http://127.0.0.1:${HTTP_PORT:-8080}/health/live" || exit 1

ENTRYPOINT ["/app/diary-service"]
