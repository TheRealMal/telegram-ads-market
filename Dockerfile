# Build stage
FROM golang:1.24-alpine AS builder

ARG TARGETARCH=amd64

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

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
COPY --from=builder /app/build/bin/server /server/bin

# Final stage
FROM scratch

# CA bundle so outbound HTTPS (e.g. api.telegram.org) verifies TLS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary from builder
COPY --from=builder /app/build/bin/server /go/bin/server

# Run the application
ENTRYPOINT ["/go/bin/server"]