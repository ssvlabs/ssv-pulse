# Use the official Golang image as a base
FROM golang:1.22@sha256:86a3c48a61915a8c62c0e1d7594730399caa3feb73655dfe96c7bc17710e96cf AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the rest of the application source code to the working directory
COPY . .

# Build the Go application
RUN CGO_ENABLED=0 go build  -o bin/benchmark cmd/benchmark/main.go

# Use a minimal base image for the final container
FROM scratch

# Set the working directory inside the container
WORKDIR /app

# Copy the binary from the builder stage to the final image
COPY --from=builder /app/bin .

# Set the entrypoint to run the binary
ENTRYPOINT ["/app/benchmark"]

# Set the default command to be empty, allowing arguments to be passed
CMD []
