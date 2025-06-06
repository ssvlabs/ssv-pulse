###############################################
FROM --platform=$BUILDPLATFORM golang:1.24@sha256:db5d0afbfb4ab648af2393b92e87eaae9ad5e01132803d80caef91b5752d289c AS builder

WORKDIR /app

COPY . .

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN go build -o bin/pulse cmd/pulse/main.go

###############################################
FROM busybox AS busybox
RUN addgroup -S pulsegroup && adduser -S -G pulsegroup pulseuser

###############################################
FROM debian:stable-slim AS certs
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