FROM golang:1.24 AS builder

# Enable Go modules and set working directory
WORKDIR /app

# Copy go mod/sum and download dependencies first (cached layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o qbcli ./main.go

# You may want to pin down a specific version (e.g. v.3.40.0)
FROM qmcgaw/gluetun:latest

COPY --from=builder /app/qbcli /usr/local/bin/qbcli
RUN chmod +x /usr/local/bin/qbcli

RUN mkdir -p  /var/cache/qbcli
VOLUME /var/cache/qbcli

ENV QBCLI_CACHE="/var/cache/qbcli" \
    QBCLI_HOST_URL="" \
    QBCLI_USERNAME="" \
    QBCLI_PASSWORD=""
