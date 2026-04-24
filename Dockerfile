FROM golang:1.26-bookworm AS builder
RUN apt-get update && apt-get install -y --no-install-recommends \
	libvirt-dev \
	libxml2-dev \
	pkg-config \
	gcc \
	&& rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=1
RUN go mod tidy \
	&& go build -o /out/api ./cmd/api \
	&& go build -o /out/worker ./cmd/worker

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
	ca-certificates \
	libvirt0 \
	qemu-utils \
	cloud-image-utils \
	genisoimage \
	xorriso \
	&& rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /out/api /usr/local/bin/api
COPY --from=builder /out/worker /usr/local/bin/worker
ENV HTTP_ADDR=:8080
