# ===========================================================================
# Stage 1: Build the Go binary
# ===========================================================================
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /splitter .

# ===========================================================================
# Stage 2: Minimal runtime image
# ===========================================================================
FROM alpine:3.21

RUN apk add --no-cache \
    tor \
    haproxy \
    privoxy \
    ca-certificates \
    curl \
    && rm -rf /var/cache/apk/*

COPY --from=builder /splitter /usr/local/bin/splitter
COPY templates/ /splitter/templates/
COPY configs/ /splitter/configs/
COPY entrypoint.sh /splitter/entrypoint.sh

RUN addgroup -S splitter 2>/dev/null || true && \
    adduser -S -G splitter -H -h /splitter splitter 2>/dev/null || true && \
    mkdir -p /tmp/splitter && \
    chown -R splitter:splitter /tmp/splitter /splitter && \
    chmod +x /splitter/entrypoint.sh

WORKDIR /splitter

HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -sf http://localhost:63540/status || exit 1

EXPOSE 63536 63537 63540

USER splitter

ENTRYPOINT ["/splitter/entrypoint.sh"]
CMD []
