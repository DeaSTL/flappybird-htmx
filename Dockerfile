# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies for CGO
RUN apk --no-cache add gcc musl-dev

WORKDIR /app

# Copy go mod files first for better caching
COPY ./app/go.mod ./app/go.sum ./
RUN go mod download

# Copy source code
COPY ./app .

# Build the binary with optimizations
# Using cache mounts for faster rebuilds
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux go build -installsuffix cgo -o flappybird .

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests and sqlite
RUN apk --no-cache add ca-certificates sqlite

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/flappybird .

# Copy static assets and templates
COPY --from=builder /app/local ./local
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/db ./db

EXPOSE 3200

CMD ["./flappybird"]
