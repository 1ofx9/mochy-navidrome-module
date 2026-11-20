FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/mochy-navidrome-module ./cmd/module

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/mochy-navidrome-module /app/mochy-navidrome-module

EXPOSE 8090
USER 1000:1000

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD ["/app/mochy-navidrome-module", "-healthcheck"]
ENTRYPOINT ["/app/mochy-navidrome-module"]
