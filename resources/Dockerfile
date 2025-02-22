FROM golang:latest AS setup
WORKDIR /app

COPY go.* ./
COPY cmd cmd
COPY internal internal

RUN go mod download && go mod verify

RUN CGO_ENABLED=0 GOOS=linux go build -o /application cmd/songlinkr/main.go

FROM alpine:latest AS compressor

RUN apk add --no-cache upx

COPY --from=setup

FROM scratch AS package

COPY --from=setup /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=setup /application /application

CMD ["/application"]
