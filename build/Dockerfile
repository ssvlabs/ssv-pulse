###############################################
FROM --platform=$BUILDPLATFORM golang:1.26@sha256:6c2a5538f964f1c82f97ad14988bf05de100d922d159d0e398b54c7b0ca0c6c9 AS builder

WORKDIR /app

COPY . .

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN go build -o bin/pulse cmd/pulse/main.go

###############################################
FROM busybox AS busybox
RUN addgroup -S pulsegroup && adduser -S -G pulsegroup pulseuser

###############################################
FROM debian:stable-slim@sha256:50db38a20a279ccf50761943c36f9e82378f92ef512293e1239b26bb77a8b496 AS certs
RUN apt-get update && apt-get install -y ca-certificates

###############################################
FROM scratch

WORKDIR /app

COPY --from=builder /app/bin/pulse .
COPY --from=builder /app/configs/config.yaml ./config.yaml

COPY --from=busybox /etc/passwd /etc/passwd
COPY --from=busybox /etc/group /etc/group

COPY --from=certs /etc/ssl/certs /etc/ssl/certs
COPY --from=certs /etc/ssl/private /etc/ssl/private
COPY --from=certs /usr/share/ca-certificates /usr/share/ca-certificates

USER pulseuser:pulsegroup

ENTRYPOINT ["/app/pulse"]
CMD []