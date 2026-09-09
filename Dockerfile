# syntax=docker/dockerfile:1

FROM golang:1.27-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY migrations/ ./migrations/

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S masonic && adduser -S masonic -G masonic

COPY --from=builder /out/server /server

USER masonic
EXPOSE 8080

ENTRYPOINT ["/server"]