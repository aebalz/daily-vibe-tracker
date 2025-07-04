# Stage 1: Build the Go application
FROM golang:1.23.4-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o /app/server ./cmd/server

# Optional: Debug build output
RUN ls -l /app/server && file /app/server

# Stage 2: Final image
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/server /app/server

# Ensure it's executable
RUN chmod +x /app/server

COPY config.env /app/config.env

EXPOSE 8080

CMD ["/app/server"]
