FROM golang:1.21 AS builder
WORKDIR /app
COPY . .
RUN go build -o ksm-scim ./cmd/main.go

FROM debian:stable-slim
RUN apt-get update && \
    apt-get install -y ca-certificates && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/ksm-scim /usr/local/bin/
CMD ["/usr/local/bin/ksm-scim"]
