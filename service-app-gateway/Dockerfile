# Stage 1: Build the application
FROM golang:1.25-alpine AS builder

WORKDIR /app/service-app-gateway

COPY service-app-gateway/go.mod service-app-gateway/go.sum ./
RUN go mod download

COPY service-app-gateway/. .

RUN CGO_ENABLED=0 GOOS=linux go build -o service-app-gateway .

# Stage 2: Create the final lightweight image
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /root/

COPY --from=builder /app/service-app-gateway/service-app-gateway .

EXPOSE 8090

CMD ["./service-app-gateway"]
