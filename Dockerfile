# ====== 1. Frontend Build Stage ======
FROM node:20-alpine AS frontend-builder

WORKDIR /app

# Copy frontend package manifests
COPY frontend/package.json frontend/package-lock.json* ./

# Install dependencies
RUN npm ci || npm install

# Copy the rest of the frontend source
COPY frontend/ ./

# Set Vite profile for production build
ENV VITE_PROFILE=prod

# Build the Vite React app
RUN npm run build

# ====== 2. Backend Build Stage ======
FROM golang:1.25.3-alpine AS backend-builder

RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

# Copy go.mod and go.sum first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy all source files
COPY . .

# Copy the built frontend static files from the previous stage into the embedded directory
# Create ui/dist just in case it doesn't exist to avoid missing directory errors
RUN mkdir -p ui/dist
COPY --from=frontend-builder /app/dist ./ui/dist/

# Build binary
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o server ./cmd/main.go

# ====== 3. Runtime stage ======
FROM alpine:latest

# Install ffmpeg + certs for HTTPS calls
RUN apk add --no-cache ffmpeg ca-certificates

WORKDIR /root/

# Copy built binary from builder
COPY --from=backend-builder /app/server .

# Expose your Gin port
EXPOSE 8080

# Run the app
CMD ["./server"]
