# ---------- Builder ----------
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build‑time dependencies (git, ca‑certificates, etc.)
RUN apk add --no-cache git ca-certificates && update-ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /lumo-api ./cmd/server

# ---------- Runtime ----------
FROM alpine:3.19

# Create a non‑root user for security
RUN addgroup -S lumo && adduser -S -G lumo lumo
USER lumo

# Copy the compiled binary
COPY --from=builder /lumo-api /usr/local/bin/lumo-api

# Expose the API port (configurable via $PORT)
EXPOSE 8080

# Entrypoint
ENTRYPOINT ["/usr/local/bin/lumo-api"]
