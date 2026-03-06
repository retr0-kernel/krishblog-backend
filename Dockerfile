# ── Stage 1: build ─────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# install ent
RUN go install entgo.io/ent/cmd/ent@latest

# generate ent code
RUN ent generate ./ent/schema

# build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-s -w" \
    -o server \
    ./cmd/server


# ── Stage 2: runtime ───────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# copy binary from builder
COPY --from=builder /app/server .

# Railway will inject PORT automatically
EXPOSE 8080

# start the server
CMD ["./server"]