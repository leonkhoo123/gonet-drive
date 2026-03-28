# ====== 1. Build stage ======
FROM node:20-alpine AS builder

WORKDIR /app

# Install dependencies
COPY package*.json ./
RUN npm ci

# Copy the rest of the source
COPY . .

# Set Vite profile for production build
ENV VITE_PROFILE=prod

# Build for production
RUN npm run build

# ====== 2. Serve stage ======
FROM nginx:stable-alpine

# Copy build output
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy custom nginx config (optional)
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Set environment variable for Nginx
ENV PORT=80

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
