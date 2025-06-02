# ---- Build stage ----
FROM golang:1.24 AS builder

ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
  -ldflags="-s -w \
  -X 'github.com/gstos/qbcli/internal/qb/version.Version=${VERSION}' \
  -X 'github.com/gstos/qbcli/internal/qb/version.Commit=${COMMIT}' \
  -X 'github.com/gstos/qbcli/internal/qb/version.Date=${DATE}'" \
  -o qbcli ./main.go

# ---- Final image ----
# TODO: Using latest to include PRs that were deployed prior to a version release
ARG GLUETUN_VERSION=latest
FROM qmcgaw/gluetun:${GLUETUN_VERSION}

COPY --from=builder --chmod=755 /app/qbcli /usr/local/bin/qbcli

RUN mkdir -p /var/cache/qbcli
VOLUME /var/cache/qbcli

ENV QBCLI_CACHE_DIR="/var/cache/qbcli" \
    QBCLI_HOST_URL="" \
    QBCLI_USERNAME="" \
    QBCLI_PASSWORD=""