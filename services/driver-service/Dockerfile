# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY services/driver-service/go.mod services/driver-service/go.sum* ./
RUN go mod download

COPY services/driver-service/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/main .

EXPOSE 8084 9094

CMD ["./main"]
