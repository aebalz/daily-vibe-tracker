# Stage 1: Build the Go application
FROM golang:1.22-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# We want to populate the module cache based on the go.mod file first.
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go application
# -o /app/server: output path for the compiled binary
# -ldflags="-w -s": strip debugging information to reduce binary size
# ./cmd/server: path to the main package
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o /app/server ./cmd/server

# Stage 2: Create the final lightweight image
FROM alpine:latest

# Add ca-certificates in case your app needs to make HTTPS requests
RUN apk --no-cache add ca-certificates

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the built Go application binary from the builder stage
COPY --from=builder /app/server /app/server

# Copy the config.env file
# This file should be present in the same directory as the Dockerfile during the build
# Or mounted via docker-compose
COPY config.env /app/config.env

# Expose the port the application runs on (as defined in config.env, default 8080)
# This is documentation, the actual port mapping is done in docker-compose or `docker run -p`
EXPOSE 8080

# Command to run the executable
# The server binary will read config.env from its working directory /app
CMD ["/app/server"]
