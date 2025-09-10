# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/k8s-resource-collector ./cmd

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/bin/k8s-resource-collector /usr/local/bin/k8s-resource-collector

# Create output directories
RUN mkdir -p /app/output /app/output-custom && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Set default command
ENTRYPOINT ["k8s-resource-collector"]

# Default arguments
CMD ["--help"]
