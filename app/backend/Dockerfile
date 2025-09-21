# Multi-stage build for Go backend
# 1) Builder
FROM golang:1.23-alpine AS builder
WORKDIR /app

# Install build tools
RUN apk add --no-cache git build-base

# Cache go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the backend source
COPY . .

# Build the server binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/server ./cmd/server

# 2) Runtime (distroless static, nonroot)
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /

# Copy binary
COPY --from=builder /bin/server /server

# Expose default port (override via env if needed)
ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["/server"]
