# ── Stage 1: build ────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Download dependencies first (cached unless go.mod/go.sum change)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a statically-linked binary.
# Migrations are embedded via embed.FS — no extra files needed at runtime.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /api ./cmd/api

# ── Stage 2: runtime ──────────────────────────────────────────────────────────
FROM alpine:3.21

# ca-certificates: needed for TLS to external services (Mercado Pago, etc.)
# tzdata: needed for correct Brazil timezone handling
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /api ./api

EXPOSE 8080

ENTRYPOINT ["./api"]
