# ── Build stage ────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy only go.mod first; generate go.sum inside the container so the
# repository can be cloned without a pre-committed go.sum file.
COPY go.mod ./
RUN go mod download -x

# Now copy the rest of the source and build
COPY . .
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bin/server ./cmd/server

# ── Runtime image ──────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /bin/server .

EXPOSE 8080
ENTRYPOINT ["/app/server"]
