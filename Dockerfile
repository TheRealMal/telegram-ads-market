# Build stage
FROM golang:1.24-alpine AS builder

ARG TARGETARCH=amd64

# Install build dependencies and wget for TON global config download
RUN apk add --no-cache git ca-certificates tzdata wget

# Set working directory
WORKDIR /app

# Download TON global config files (used by liteclient at runtime; avoids runtime network fetch)
RUN mkdir -p /app/config \
	&& wget -q -O /app/config/global-config.json "https://ton.org/global-config.json" \
	&& wget -q -O /app/config/testnet-global.config.json "https://ton-blockchain.github.io/testnet-global.config.json"

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg

# Build the application
# COPY build/bin/server ./build/bin/server
RUN env CGO_ENABLED=0 GOARCH=${TARGETARCH} go build -o ./build/bin/server -ldflags '-s' ./cmd/main.go

# Artifact stage: export binary to host with:
#   DOCKER_BUILDKIT=1 docker build --output type=local,dest=./build/bin/ --target=artifact .
# Binary is written to ./build/server
FROM scratch AS artifact
COPY --from=builder /app/build/bin/server /server

# Final stage
FROM scratch

# CA bundle so outbound HTTPS (e.g. api.telegram.org) verifies TLS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# TON global config (downloaded at build); set LITECLIENT_GLOBAL_CONFIG_DIR=/etc/ton to use
COPY --from=builder /app/config /etc/ton

# Copy binary from builder
COPY --from=builder /app/build/bin/server /go/bin/server

# Run the application
ENTRYPOINT ["/go/bin/server"]