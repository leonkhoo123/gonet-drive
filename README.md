# GoNet Drive

GoNet Drive is a high-performance, self-hosted file management and media streaming server. It combines a robust backend written in Go with a modern, responsive frontend built with React, Vite, and Tailwind CSS. The entire application is compiled into a single executable binary (which embeds the frontend), making deployment incredibly simple, whether running directly on the host or inside a Docker container.

## Features

- 📁 **File Management**: Upload (including chunking smaller than 100mb & drag-and-drop), download, move, copy, rename, delete, and check for duplicate files.
- 🎬 **Video Streaming**: Built-in video player with streaming support.
- 🎵 **Music Player**: Listen to your music right in the browser with full queue and playlist support.
- 📚 **Audiobook Player**: Dedicated player for audiobooks that automatically remembers your playback progress across sessions.
- 📄 **Document Viewing**: View PDF files and plain text documents directly in the web UI without downloading them first.
- 🔗 **Secure File Sharing**: Create shareable links for files and folders. You can set optional passwords, expiration dates, and manage all your active shared links from an admin interface.
- 🔐 **Security & Access**:
  - Every critical endpoint is protected by secured JWTs using HTTP-only cookies.
  - Multi-Factor Authentication (eg.Google Authenticator) support.
  - API rate-limiting to prevent brute-force attacks (e.g., password or MFA guessing).
- 📱 **Modern UI**: Dark/Light mode, responsive design that works beautifully on desktop and mobile devices (PWA supported).
- 🚀 **Easy Deployment**: A multi-stage Dockerfile builds a single binary output with frontend fully embedded.

## Tech Stack

- **Backend**: Go 1.24+, Gin Framework, SQLite (for metadata & users), Gorilla WebSockets.
- **Frontend**: React 18+, TypeScript, Vite, Tailwind CSS, shadcn/ui.

## Installation & Running

### Using Docker (Recommended)

*Note: There is currently no pre-built Docker image repository available. You will need to clone this repository and build the image manually.*

The easiest way to get started is by using Docker. The provided `Dockerfile` will build both the frontend and backend, and run the final lightweight Alpine image.

```bash
# Clone the repository
git clone https://github.com/leonkhoo123/gonet-drive.git
cd gonet-drive

# Build the Docker image
docker build -t gonet-drive .

# Run the container
# This exposes the web UI on port 8080.
# Make sure to mount volumes for both your files (bind mount) and the database (docker volume) to persist them.
docker run -d \
  --name gonet-drive \
  -p 8080:8080 \
  -e APP_JWTSECRET="<your_jwt_secret_key>" \
  -e ADMIN_USER="<admin_username>" \
  -e ADMIN_PASS="<admin_secure_password>" \
  -v /path/to/your/files:/app/data \
  -v gonet_db:/app/db \
  gonet-drive
```

## Environment Variables

The server can be configured using environment variables. You can set them directly in your environment or use a `.env` file in the directory where the binary is executed.

- `WORK_DIR`: The root directory for the files to serve and manage. *(Default: `/app/data`)*
- `DB_DIR`: The directory where the SQLite metadata database is stored. *(Default: `/app/db`)*
- `LISTEN_ADDR`: The host and port for the server to listen on. *(Default: `:8080`)*
- `APP_JWTSECRET`: The secret key used to sign JWT auth tokens. **(Required)**
- `ADMIN_USER`: The username for the initial superadmin account. *(Used only on first run to bootstrap the database)*
- `ADMIN_PASS`: The password for the initial superadmin account. *(Used only on first run to bootstrap the database)*
- `VIDEO_HOSTNAME`: Optional public URL override to serve video streams.
- `ALLOWED_ORIGINS`: Comma-separated list of allowed CORS origins (useful for local development).
- `DEFAULT_SERVICE_NAME`: The name of your cloud server displayed in the UI. *(Default: `My Cloud Server`)*

## Project Structure

- `cmd/main.go`: The main entrypoint for the Go application.
- `internal/`: Contains the Go backend logic (controllers, services, repositories, database models, etc.).
- `database/migrations/`: SQL migration files for the SQLite database.
- `frontend/`: The React + Vite frontend source code.
- `ui/`: Holds the embedded frontend `dist/` folder after compilation, injected into the Go binary.

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.
