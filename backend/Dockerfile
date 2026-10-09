FROM golang:1.27-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux \
    go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/rtsp-stream-console \
    ./cmd/server


FROM alpine:3.22

RUN apk add --no-cache \
    ca-certificates \
    ffmpeg \
    && addgroup -S app \
    && adduser -S -G app app

WORKDIR /app

COPY --from=builder \
    /out/rtsp-stream-console \
    /usr/local/bin/rtsp-stream-console

USER app

EXPOSE 8080

ENV HTTP_ADDR=:8080

ENTRYPOINT ["rtsp-stream-console"]