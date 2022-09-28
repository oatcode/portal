## Build
FROM golang:1.17-alpine AS builder
WORKDIR /build
COPY . ./
RUN go build -o onprem.bin

## Deploy
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /build/onprem.bin ./
EXPOSE 8080
ENTRYPOINT ["./onprem.bin"]
