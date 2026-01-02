# Build stage
FROM golang:1.21-alpine AS build
WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o settlement-service ./cmd/server

# Runtime stage
FROM alpine:3.19
WORKDIR /app

RUN apk --no-cache add ca-certificates && \
    addgroup -g 1001 -S appgroup && \
    adduser -S appuser -u 1001 -G appgroup

COPY --from=build /app/settlement-service .

USER appuser

EXPOSE 8083 9093

CMD ["./settlement-service"]
