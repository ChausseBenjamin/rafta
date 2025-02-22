FROM golang:latest AS compile
WORKDIR /build

COPY . .
RUN go mod download && go mod verify

# Enable static linking for scratch
RUN CGO_ENABLED=0 GOOS=linux go build -o /application -ldflags="-s -w" main.go

FROM alpine:latest AS compressor
RUN apk add --no-cache upx
COPY --from=compile /application /application
RUN upx --best /application

FROM scratch AS package

# Copy CA certificates
COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=compressor /application /application

ENTRYPOINT ["/application"]
