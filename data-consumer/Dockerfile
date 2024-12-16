# Stage 1: Build the Go app
FROM golang:1.23.1 AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Go Modules manifests
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod tidy

# Copy the source code into the container
COPY . .

# Build the Go app (no cross-compilation to start)
RUN go build -o data-consumer main.go

# Run tests (optional, depending on your needs)
RUN go test -v

# Stage 2: Create the final smaller image for running the app
FROM golang:1.23.1

# Install necessary dependencies for running the app (like certificates for SSL, etc.)
RUN apt-get update && apt-get install -y ca-certificates

# Set the Current Working Directory inside the container
WORKDIR /root/

# Copy the pre-built binary from the builder stage
COPY --from=builder /app/data-consumer .

# Ensure the Go application logs to stdout and stderr
CMD ["./data-consumer"]

