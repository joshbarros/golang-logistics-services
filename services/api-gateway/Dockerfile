# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY services/api-gateway/go.mod services/api-gateway/go.sum* ./
RUN go mod download

COPY services/api-gateway/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/main .

EXPOSE 8000

CMD ["./main"]
