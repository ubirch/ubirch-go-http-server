FROM golang:1.13 AS builder
COPY . /app
WORKDIR /app/main
RUN \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOPROXY=https://proxy.golang.org,direct \
    go build -o main .


FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app/main /main
ENTRYPOINT ["/main"]
CMD ["server"]