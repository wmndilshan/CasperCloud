FROM golang:1.23 AS builder
WORKDIR /app
COPY go.mod .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/worker ./cmd/worker

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates libvirt-clients cloud-image-utils && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /out/api /usr/local/bin/api
COPY --from=builder /out/worker /usr/local/bin/worker
ENV HTTP_ADDR=:8080
