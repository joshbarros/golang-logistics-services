# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY services/route-service/go.mod services/route-service/go.sum* ./
RUN go mod download

COPY services/route-service/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/main .

EXPOSE 8083 9093

CMD ["./main"]
