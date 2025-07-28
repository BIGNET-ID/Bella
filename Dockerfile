# --- STAGE 1: Build Stage ---
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache ca-certificates git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bella ./cmd/main.go


# --- STAGE 2: Runtime Stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates
WORKDIR /app

COPY --from=builder /bella .

ENTRYPOINT ["./bella"]
