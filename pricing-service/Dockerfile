# Image size 70mb
# Stage 1: Build the application
FROM golang:1.25-alpine AS builder

WORKDIR /app/pricing-service # Work inside the service directory

# Copy go.mod and go.sum first to leverage Docker cache for dependencies
COPY pricing-service/go.mod pricing-service/go.sum ./
RUN go mod download

# Copy the rest of the source code for pricing-service
COPY pricing-service/. .

# Build the binary. CGO_ENABLED=0 ensures a statically linked binary.
RUN CGO_ENABLED=0 GOOS=linux go build -o pricing-service .

# Stage 2: Create the final lightweight image
FROM alpine:latest

# Install CA certificates required for HTTPS calls (e.g., fetching JWKS)
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/pricing-service/pricing-service .

# Expose the port the app runs on (default for Gin is 8080)
EXPOSE 8091

CMD ["./pricing-service"]
