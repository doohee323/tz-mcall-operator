# Build stage
FROM golang:1.23-alpine AS builder

# Install necessary packages
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /workspace

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the controller
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -o manager \
    ./cmd/controller/main.go

# Final stage
FROM alpine:3.18

# Install necessary packages (including bash and coreutils for proper command execution)
RUN apk add --no-cache ca-certificates tzdata wget bash coreutils

# Create non-root user
RUN addgroup -g 1000 mcall && \
    adduser -D -u 1000 -G mcall mcall

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /workspace/manager .

# Set executable permissions
RUN chmod +x manager

# Copy CRD definitions (templates)
COPY --from=builder /workspace/helm/mcall-operator/templates/crds/ ./crd/

# Copy examples
COPY --from=builder /workspace/examples/ ./examples/

# Copy Helm chart
COPY --from=builder /workspace/helm/ ./helm/

# Change ownership to non-root user
RUN chown -R mcall:mcall /app

# Switch to non-root user
USER mcall

# Expose ports
EXPOSE 8080 8081 8443

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/healthz || exit 1

# Run the controller
ENTRYPOINT ["./manager"]
