# Stage 1: Build binary
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install certificates for HTTPS requests
RUN apk add --no-cache ca-certificates

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build static binary with size optimizations
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-w -s" \
    -o /app/server \
    ./cmd/server/main.go

# Stage 2: Lightweight Alpine runtime with ffmpeg
FROM alpine:3.20

RUN apk add --no-cache ca-certificates ffmpeg tzdata

WORKDIR /app

COPY --from=builder /app/server /app/server

# Create nonroot user for security
RUN adduser -D -u 10001 nonroot
USER nonroot:nonroot

ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["/app/server"]
