# Build stage
FROM golang:1.21-alpine AS builder

# Install git and build dependencies
RUN apk add --no-cache git build-base

# Set working directory
WORKDIR /app

# Copy the entire project
COPY . .

# Download dependencies
RUN go mod download
RUN go mod verify

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o api-gateway ./api-gateway/main.go

# Final stage
FROM alpine:latest

# Install necessary runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the binary and config
COPY --from=builder /app/api-gateway .
COPY --from=builder /app/config.yaml /etc/microservices/

# Set environment variables
ENV CONFIG_FILE=/etc/microservices/config.yaml

# Expose HTTP and metrics ports
EXPOSE 8080 9091

# Set entry point
CMD ["./api-gateway"]
