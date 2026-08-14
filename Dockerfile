# syntax=docker/dockerfile:1

ARG GO_VERSION

FROM golang:${GO_VERSION}-alpine AS builder
ARG VERSION=dev
ARG REVISION=dev

RUN echo 'nobody:x:65534:65534:Nobody:/:' > /tmp/passwd && \
    apk add --no-cache upx

WORKDIR /src

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.Version=${VERSION} -X main.Gitsha=${REVISION}" \
    ./cmd/dexd && upx --best --lzma dexd

FROM scratch
COPY --from=builder /tmp/passwd /etc/passwd
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder --chmod=555 /src/dexd /dexd

ENTRYPOINT ["/dexd"]
