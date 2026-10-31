FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY tecnologia/livebid-backend/go.mod ./
RUN go mod download
COPY tecnologia/livebid-backend ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/livebid-server ./cmd/server

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/livebid-server /app/livebid-server
RUN adduser -D -u 1000 livebiduser
USER livebiduser

EXPOSE 3000
ENV PORT=3000
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3000/health || exit 1
ENTRYPOINT ["/app/livebid-server"]
