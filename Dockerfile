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
FROM golang:1.25.10-alpine AS backend-builder

RUN apk add --no-cache git gcc musl-dev vips-dev

WORKDIR /app

# GOPRIVATE tells Go this module path is private (skip proxy, go direct to git)
ENV GOPRIVATE=github.com/leonkhoo123
# GONOSUMCHECK skips the public checksum database for private modules
ENV GONOSUMCHECK=github.com/leonkhoo123
# GONOSUMDB skips sum.golang.org for private modules
ENV GONOSUMDB=github.com/leonkhoo123

# Configure git to authenticate with GitHub token for private repos
ARG GITHUB_TOKEN
RUN git config --global url."https://${GITHUB_TOKEN}:x-oauth-basic@github.com/".insteadOf "https://github.com/"

# Copy go manifests first for layer caching
COPY go.mod go.sum ./

# Remove local replace directive so Go fetches the private module from GitHub
RUN go mod edit -dropreplace github.com/leonkhoo123/gonet-auth

# Cache Go modules
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Reset git config (don't leak token in image layers)
RUN git config --global --unset url."https://${GITHUB_TOKEN}:x-oauth-basic@github.com/".insteadOf

# Copy source last
COPY . .

# Ensure embed directory exists before copying frontend dist
RUN mkdir -p ui/dist
COPY --from=frontend-builder /app/dist ./ui/dist/

# Build with BuildKit cache for Go build cache + modules, strip debug info
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go mod edit -dropreplace github.com/leonkhoo123/gonet-auth && \
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -trimpath -o server ./cmd/main.go

# ====== 3. Runtime stage ======
FROM alpine:latest

RUN echo -e "https://dl-cdn.alpinelinux.org/alpine/edge/main\nhttps://dl-cdn.alpinelinux.org/alpine/edge/community" >> /etc/apk/repositories && \
    apk add --no-cache ffmpeg=8.1.1-r0 vips && \
    sed -i '/edge/d' /etc/apk/repositories && \
    apk add --no-cache util-linux ca-certificates

WORKDIR /root/

COPY --from=backend-builder /app/server .

EXPOSE 8080

CMD ["./server"]
