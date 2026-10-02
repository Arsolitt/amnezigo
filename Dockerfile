FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ./build/amnezigo ./cmd/amnezigo/

FROM amneziavpn/amneziawg-go:3.1.20260828

RUN apk --no-cache add ca-certificates bash

COPY --from=builder /app/build/amnezigo /usr/local/bin/amnezigo
