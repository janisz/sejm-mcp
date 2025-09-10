# Build stage
FROM golang:1.23-alpine AS builder

# Install git and ca-certificates (needed for go modules and HTTPS)
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build static binary with no build tags
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o sejm-mcp ./cmd/sejm-mcp

# Final stage - use scratch for minimal image
FROM scratch

# Copy ca-certificates from builder for HTTPS requests to Polish Parliament APIs
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the static binary from builder stage
COPY --from=builder /app/sejm-mcp /sejm-mcp

# The MCP server communicates via stdio, so no ports need to be exposed
# EXPOSE is not needed for MCP servers

# Run the MCP server
ENTRYPOINT ["/sejm-mcp"]