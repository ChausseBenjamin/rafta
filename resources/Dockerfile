FROM alpine:latest AS zig_download
RUN apk update && \
  apk add \
  curl xz
ARG ZIGVER=0.14.0
ARG ZIGPKG=zig-linux-x86_64-$ZIGVER
ARG ZIGURL=https://ziglang.org/download/${ZIGVER}/${ZIGPKG}.tar.xz
RUN mkdir -p /deps
WORKDIR /deps

RUN curl -L ${ZIGURL} -O && \
  tar xf ${ZIGPKG}.tar.xz && \
  mv ${ZIGPKG}/ local/

## Prepare Go image
FROM golang:1.24-alpine as compile
RUN apk --no-cache add \
  libc-dev \
  xz \
  samurai \
  git \
  upx \
  cmake

COPY --from=zig_download /deps/local/ /deps/local/
RUN ln -s /deps/local/zig /usr/local/bin

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download && go mod verify

ENV CGO_ENABLED=1
ENV GOOS=linux
ENV CC="zig cc -target x86_64-linux"

COPY external ./external/
COPY internal ./internal/
COPY pkg ./pkg/
COPY main.go ./
RUN go build -o /app -ldflags="-v -w -s -linkmode=external" .

RUN upx --overlay=strip --best /app

FROM scratch as package

COPY --from=compile /app /app

ENTRYPOINT [ "/app" ]

