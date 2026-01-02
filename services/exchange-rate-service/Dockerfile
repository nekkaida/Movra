# Build stage
FROM golang:1.21-alpine AS build
WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o exchange-rate-service ./cmd/server

# Runtime stage
FROM alpine:3.19
WORKDIR /app

# Add CA certificates and create non-root user
RUN apk --no-cache add ca-certificates && \
    addgroup -g 1001 -S appgroup && \
    adduser -S appuser -u 1001 -G appgroup

COPY --from=build /app/exchange-rate-service .

USER appuser

EXPOSE 8082 9092

CMD ["./exchange-rate-service"]
