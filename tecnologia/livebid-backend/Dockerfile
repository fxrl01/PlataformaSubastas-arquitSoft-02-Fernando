# ==============================================================================
# LIVEBID BACKEND — DOCKERFILE MULTI-STAGE (AWS FREE TIER OPTIMIZED)
# Tamaño final de imagen: ~18 MB | Consumo RAM: ~20 MB
# ==============================================================================

# 1. ETAPA DE CONSTRUCCIÓN (BUILDER)
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Instalar certificados CA y tzdata
RUN apk add --no-cache ca-certificates tzdata

# Copiar manifiesto de dependencias
COPY go.mod ./
RUN go mod download

# Copiar código fuente
COPY . .

# Compilar binario estático optimizado (-ldflags: -s -w elimina símbolos de depuración)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /app/livebid-server ./cmd/server/main.go

# 2. ETAPA FINAL DE PRODUCCIÓN (IMAGEN MINIMALISTA SCRATCH / ALPINE)
FROM alpine:3.20

WORKDIR /app

# Copiar certificados y timezone desde el builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/livebid-server /app/livebid-server

# Usuario sin privilegios por seguridad
RUN adduser -D -u 1000 livebiduser
USER livebiduser

EXPOSE 8080

ENV PORT=8080

HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/livebid-server"]

