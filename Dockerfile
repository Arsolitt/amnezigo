FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ./build/amnezigo ./cmd/amnezigo/

FROM amneziavpn/amneziawg-go:0.2.16

RUN apk --no-cache add ca-certificates bash

COPY --from=builder /app/build/amnezigo /usr/local/bin/amnezigo
