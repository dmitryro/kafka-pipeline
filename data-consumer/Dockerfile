# Use the latest  Go image
FROM golang:1.23.4

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the application code
COPY . .

# Build the Go application
RUN go build -o data-consumer main.go

# Run the Go application
CMD ["./data-consumer"]
