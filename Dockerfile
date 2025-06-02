# ------------------------------
# STAGE 1: Build
# ------------------------------
FROM golang:1.24 AS builder

# Enable Go modules and set working directory
WORKDIR /app

# Copy go mod/sum and download dependencies first (cached layer)
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o qbcli ./main.go

# ------------------------------
# STAGE 2: Runtime (minimal image)
# ------------------------------
FROM alpine:latest

# Copy the compiled binary from the builder stage
COPY --from=builder /app/qbcli /usr/local/bin/qbcli
RUN chmod +x /usr/local/bin/qbcli

# Create a directory structure for caching session cookies
RUN mkdir -p  /var/cache/qbcli

# Set ownership and permissions (optional)
RUN adduser -D -g '' appuser && \
    chown appuser /usr/local/bin/qbcli && \
    chown appuser /var/cache/qbcli

USER appuser

VOLUME /var/cache/qbcli

ENV QBCLI_CACHE="/var/cache/qbcli" \
    QBCLI_HOST_URL="" \
    QBCLI_USERNAME="" \
    QBCLI_PASSWORD=""

ENTRYPOINT ["qbcli"]
