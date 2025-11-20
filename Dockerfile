FROM golang:1.21 AS builder
WORKDIR /app
COPY . .
RUN go build -o ksm-scim ./cmd/main.go

FROM debian:stable-slim
COPY --from=builder /app/ksm-scim /usr/local/bin/
CMD ["/usr/local/bin/ksm-scim"]
