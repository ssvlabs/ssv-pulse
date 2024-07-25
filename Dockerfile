# Use the official Golang image as a base
FROM golang:1.22.5 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the go.mod and go.sum files to the working directory
COPY go.mod go.sum ./

# Download the dependencies
RUN go mod download

# Copy the rest of the application source code to the working directory
COPY . .

# Build the Go application
RUN go build -o /app/main .

# Use a minimal base image for the final container
FROM debian:stable-slim

# Set the working directory inside the container
WORKDIR /app

# Copy the binary from the builder stage to the final image
COPY --from=builder /app/main .

# Set the command to run the binary
CMD ["/app/main"]
