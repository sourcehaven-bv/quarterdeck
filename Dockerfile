FROM golang:1.22-alpine AS builder

WORKDIR /build

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build args for version info
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

# Build static binary
RUN CGO_ENABLED=0 go build \
    -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o quarterdeck .

# Final minimal image
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/quarterdeck /quarterdeck

ENTRYPOINT ["/quarterdeck"]
