###############################################
FROM golang:1.23@sha256:613a108a4a4b1dfb6923305db791a19d088f77632317cfc3446825c54fb862cd AS builder

WORKDIR /app

COPY . .
RUN CGO_ENABLED=0 go build -o bin/benchmark cmd/benchmark/main.go

###############################################
FROM busybox AS busybox
RUN addgroup -S benchmarkgroup && adduser -S -G benchmarkgroup benchmarkuser

###############################################
FROM debian:stable-slim AS certs
RUN apt-get update && apt-get install -y ca-certificates

###############################################
FROM scratch

WORKDIR /app

COPY --from=builder /app/bin/benchmark .
COPY --from=builder /app/configs/config.yaml ./config.yaml

COPY --from=busybox /etc/passwd /etc/passwd
COPY --from=busybox /etc/group /etc/group

COPY --from=certs /etc/ssl/certs /etc/ssl/certs
COPY --from=certs /etc/ssl/private /etc/ssl/private
COPY --from=certs /usr/share/ca-certificates /usr/share/ca-certificates

USER benchmarkuser:benchmarkgroup

ENTRYPOINT ["/app/benchmark"]
CMD []