# ---------- Stage 1: Build ----------
FROM golang:1.25.1-alpine3.21 AS builder

# Install CA certificates (needed if go mod downloads via HTTPS)
RUN apk add --no-cache ca-certificates git

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum first for dependency caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Declare a build argument for Git SHA
ARG GIT_SHA=unknown

# Optional: print its value to verify during build
RUN echo "Building with GIT_SHA=$GIT_SHA"

# Build static binary (disable cgo)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-X main.commit=$GIT_SHA" \
    -o server ./

# ---------- Stage 2: Runtime ----------
FROM alpine:3.21

# Add CA certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Create a non-root user for security
RUN addgroup -S app && adduser -S app -G app
USER app

# Set working directory
WORKDIR /app

# Copy built binary from builder
COPY --from=builder /app/server .

# Run the binary
ENTRYPOINT ["./server"]
