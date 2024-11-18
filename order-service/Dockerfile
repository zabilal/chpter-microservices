# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o order-service .

# Final stage
FROM alpine:latest

# Install necessary runtime dependencies
RUN apk --no-cache add ca-certificates

# Copy the binary from builder
COPY --from=builder /app/order-service /usr/local/bin/
COPY --from=builder /app/config /config

# Set environment variables
ENV CONFIG_PATH=/config

# Expose gRPC port
EXPOSE 50052

# Set entry point
ENTRYPOINT ["order-service"]