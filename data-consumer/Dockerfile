# Stage 1: Build the Go app
FROM golang:1.23.1 AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Go Modules manifests
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod tidy

# Copy the source code into the container
COPY . .

# Build the Go app (no cross-compilation to start)
RUN go build -o data-consumer main.go

# Run tests (optional)
RUN go test -v

# Stage 2: Create the final smaller image for running the app
FROM golang:1.23.1-alpine

# Install necessary dependencies for running the app (including libc6-compat)
RUN apk --no-cache add ca-certificates libc6-compat

# Set the Current Working Directory inside the container
WORKDIR /root/

# Copy the pre-built binary from the builder stage
COPY --from=builder /app/data-consumer .

# Healthcheck script
COPY healthcheck.sh /usr/local/bin/healthcheck.sh
RUN chmod +x /usr/local/bin/healthcheck.sh

# Set the Healthcheck in Dockerfile
HEALTHCHECK --interval=10s --timeout=5s --retries=5 \
  CMD /usr/local/bin/healthcheck.sh || exit 1

# Ensure the Go application logs to stdout and stderr
CMD ["./data-consumer"]
