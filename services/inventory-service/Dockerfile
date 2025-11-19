# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY services/inventory-service/go.mod services/inventory-service/go.sum* ./
RUN go mod download

COPY services/inventory-service/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/main .

EXPOSE 8082 9092

CMD ["./main"]
