# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go workspace files
COPY go.work go.work.sum* ./
COPY shared/go.mod shared/go.sum* ./shared/
COPY api/go.mod api/go.sum* ./api/

# Download dependencies
WORKDIR /app/api
RUN go mod download

# Copy source code
WORKDIR /app
COPY shared/ ./shared/
COPY api/ ./api/

# Build the binary
WORKDIR /app/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/api

# Production stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/api /app/api

EXPOSE 8080

USER nobody:nobody

ENTRYPOINT ["/app/api"]
