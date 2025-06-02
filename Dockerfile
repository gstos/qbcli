# ------------------------------
# STAGE 1: Build
# ------------------------------
FROM golang:1.24 AS builder

# Enable Go modules and set working directory
WORKDIR /app

# Copy go mod/sum and download dependencies first (cached layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source
COPY ./cmd ./cmd
COPY ./internal ./internal

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o qbcli ./main.go

# ------------------------------
# STAGE 2: Runtime (minimal image)
# ------------------------------
FROM alpine:latest

# Add a user (optional but recommended)
RUN adduser -D -g '' appuser

# Copy the compiled binary from the builder stage
COPY --from=builder /app/qbcli /usr/local/bin/qbcli

# Set ownership and permissions (optional)
RUN chown appuser /usr/local/bin/qbcli

# Use non-root user
USER appuser

# Set default command
ENTRYPOINT ["qbcli"]