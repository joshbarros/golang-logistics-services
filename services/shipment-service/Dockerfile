# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY services/shipment-service/go.mod services/shipment-service/go.sum* ./
RUN go mod download

# Copy source code
COPY services/shipment-service/ ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .

# Expose ports
EXPOSE 8081 9091

# Run the application
CMD ["./main"]
