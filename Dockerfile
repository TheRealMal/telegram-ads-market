# Build stage
FROM golang:1.24-alpine AS builder

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
RUN env CGO_ENABLED=0 go build -o ./build/bin/server -ldflags '-s' ./cmd/main.go

# Final stage
FROM scratch

# Copy binary from builder
COPY --from=builder /app/build/bin/server /go/bin/server

# Run the application
ENTRYPOINT ["/go/bin/server"]