# Stage 1: Build the application
FROM golang:1.25-alpine AS builder

WORKDIR /app/member-request-service

COPY member-request-service/go.mod member-request-service/go.sum ./
RUN go mod download

COPY member-request-service/. .

RUN CGO_ENABLED=0 GOOS=linux go build -o member-request-service .

# Stage 2: Create the final lightweight image
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /root/

COPY --from=builder /app/member-request-service/member-request-service .

EXPOSE 8084

CMD ["./member-request-service"]
