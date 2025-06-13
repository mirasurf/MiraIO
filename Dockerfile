# Build stage
FROM registry.cn-hangzhou.aliyuncs.com/lacogito/goapp-builder:1.23.8-alpine3.21 AS builder

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Copy source code
COPY . .

# Download dependencies
RUN --mount=type=cache,target=/gomod-cache go mod download

# Build the application
RUN --mount=type=cache,target=/gomod-cache \
    --mount=type=cache,target=/go-cache \
    CGO_ENABLED=0 GOOS=linux go build -o /app/miraio pkg/main.go

# Final stage
FROM registry.cn-hangzhou.aliyuncs.com/lacogito/alpine:3.21

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Set working directory
WORKDIR /app
ENV TZ="Asia/Shanghai"

# Create logs directory
RUN mkdir -p /app/logs && chmod 755 /app/logs

# Copy the binary from builder
COPY --from=builder /app/miraio .
COPY .env.development .env

# Expose the application port
EXPOSE 9080

# Run the application
CMD ["./miraio"] 