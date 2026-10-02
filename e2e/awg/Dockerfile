ARG AWG_GO_TAG=v3.1.20260828
ARG AWG_TOOLS_TAG=v3.1.20260812

FROM golang:1.26-alpine AS go-build
ARG AWG_GO_TAG
RUN apk add --no-cache git && \
    git clone --depth 1 --branch ${AWG_GO_TAG} https://github.com/amnezia-vpn/amneziawg-go /src
WORKDIR /src
RUN CGO_ENABLED=0 go build -o /out/amneziawg-go .

FROM alpine:3.19 AS tools-build
ARG AWG_TOOLS_TAG
RUN apk add --no-cache git build-base linux-headers && \
    git clone --depth 1 --branch ${AWG_TOOLS_TAG} https://github.com/amnezia-vpn/amneziawg-tools /tools && \
    make -C /tools/src

FROM alpine:3.19
RUN apk add --no-cache iproute2 iputils tcpdump
COPY --from=go-build /out/amneziawg-go /usr/bin/amneziawg-go
COPY --from=tools-build /tools/src/wg /usr/bin/awg
CMD ["sleep", "infinity"]
