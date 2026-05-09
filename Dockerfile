# syntax=docker/dockerfile:1
# ====== 1. Frontend Build Stage ======
FROM node:20-alpine AS frontend-builder

WORKDIR /app

# Copy package manifests first for layer caching
COPY frontend/package.json frontend/package-lock.json* ./

# Cache npm downloads across builds
RUN --mount=type=cache,target=/root/.npm npm ci

# Copy frontend source last (changes most often)
COPY frontend/ ./

ENV VITE_PROFILE=prod

RUN npm run build

# ====== 2. Backend Build Stage ======
FROM golang:1.25.3-alpine AS backend-builder

RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

# Copy go manifests first for layer caching
COPY go.mod go.sum ./

# Cache Go modules
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy source last
COPY . .

# Ensure embed directory exists before copying frontend dist
RUN mkdir -p ui/dist
COPY --from=frontend-builder /app/dist ./ui/dist/

# Build with BuildKit cache for Go build cache + modules, strip debug info
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -trimpath -o server ./cmd/main.go

# ====== 3. Runtime stage ======
FROM alpine:latest

RUN apk add --no-cache ffmpeg ca-certificates

WORKDIR /root/

COPY --from=backend-builder /app/server .

EXPOSE 8080

CMD ["./server"]
