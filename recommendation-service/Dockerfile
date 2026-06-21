# Stage 1: Build the application
FROM golang:1.25-alpine AS builder

WORKDIR /app/recommendation-service

COPY recommendation-service/go.mod recommendation-service/go.sum ./
RUN go mod download

COPY recommendation-service/. .

RUN CGO_ENABLED=0 GOOS=linux go build -o recommendation-service .

# Stage 2: Create the final lightweight image
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /root/

COPY --from=builder /app/recommendation-service/recommendation-service .

EXPOSE 8085

CMD ["./recommendation-service"]
