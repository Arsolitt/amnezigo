FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Version stamp baked into the binary; override with:
#   docker build --build-arg VERSION=vX.Y.Z \
#     --build-arg COMMIT=$(git rev-parse --short HEAD) -t amnezigo .
ARG VERSION=dev
ARG COMMIT=none
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X github.com/Arsolitt/amnezigo/internal/buildinfo.Version=${VERSION} -X github.com/Arsolitt/amnezigo/internal/buildinfo.Commit=${COMMIT}" \
    -o ./build/amnezigo ./cmd/amnezigo/

FROM amneziavpn/amneziawg-go:3.1.20260828

# No package layer: the base is Alpine 3.19 and already ships bash, the
# ca-certificates bundle and the awg tools, so the build needs no network
# access beyond the base image pull.

COPY --from=builder /app/build/amnezigo /usr/local/bin/amnezigo

# Same entrypoint as the published image (cmd/amnezigo/Dockerfile) so the
# documented `docker run --rm amnezigo <command>` works without --entrypoint.
# Override with `--entrypoint /bin/sh` for a shell.
ENTRYPOINT ["/usr/local/bin/amnezigo"]
