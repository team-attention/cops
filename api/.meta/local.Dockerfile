# Development stage with Air hot reload
FROM golang:1.25-alpine

WORKDIR /app

# Install air
RUN go install github.com/air-verse/air@latest

# Copy workspace files
COPY go.work ./
COPY shared/ ./shared/
COPY api/ ./api/

WORKDIR /app/api

EXPOSE 8080

CMD ["air", "-c", ".meta/.air.toml"]
