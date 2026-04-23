# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git gcc musl-dev ca-certificates
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG BINARY=api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/server ./cmd/${BINARY}

# ── Distroless run stage ────────────────────────────────────────────────────
FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app
COPY --from=builder /bin/server .
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/server"]
