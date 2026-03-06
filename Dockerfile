# ── Stage 1: build ─────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# generate ent BEFORE install to avoid extra memory
RUN go run entgo.io/ent/cmd/ent generate ./ent/schema

# build with limited parallelism (important for Railway builders)
ENV GOMAXPROCS=2

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -p 1 \
    -ldflags="-s -w" \
    -o server \
    ./cmd/server


# ── Stage 2: runtime ───────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/server .

EXPOSE 8080

CMD ["./server"]