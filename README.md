# Mochy Navidrome Module

[![CI](https://github.com/1ofx9/mochy-navidrome-module/actions/workflows/ci.yml/badge.svg)](https://github.com/1ofx9/mochy-navidrome-module/actions/workflows/ci.yml)
[![Docker](https://github.com/1ofx9/mochy-navidrome-module/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/1ofx9/mochy-navidrome-module/actions/workflows/docker-publish.yml)
[![Release](https://img.shields.io/github/v/release/1ofx9/mochy-navidrome-module?color=blue)](https://github.com/1ofx9/mochy-navidrome-module/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A lightweight, stateless web module connecting [Navidrome](https://www.navidrome.org/) and Subsonic-compatible music servers to [Mochy Music](https://realidkroo.github.io/Mochy-Music/) on Android.

---

## Features

- **Mochy Web Module Spec**: Implements `/manifest.json`, `/icon.png`, `/logo.png`, `/search`, `/stream/{id}`, `/audio/{id}`, `/art/{id}`, `/lyrics`, `/metadata`, and `/health`.
- **Client-Side Auth**: Server URL, username, and password are saved in Mochy Music on your phone and passed with requests.
- **Docker Auto-Discovery**: Automatically discovers Navidrome on the same Docker network (`http://navidrome:4533`) and auto-hides the "Server URL" box in the app.
- **Continuous Audio Streaming**: Unbounded stream proxy supporting HTTP `Range` requests and `206 Partial Content` seeking without buffer cutoffs.
- **Lossless & Hi-Res Format Badging**: Resolves audio codec, sample rate, bit depth, and player badges (`Hi-Res Lossless`, `Lossless`, `High`, `Low`).
- **Synchronized LRC Lyrics**: Fetches timed lyrics from Navidrome via OpenSubsonic `getLyricsBySongId` with fallback to standard Subsonic lyrics.
- **In-Memory Artwork Cache**: Thread-safe LRU cover art cache with SHA-256 ETags and HTTP 304 revalidation.
- **Multi-Instance Support**: Configurable `MODULE_ID` and `MODULE_NAME` to serve multiple Navidrome servers side-by-side in Mochy Music.
- **Zero Third-Party Dependencies**: Pure Go standard library implementation.

---

## Docker Deployment

### Option A: Docker Compose (Recommended)

Copy [`compose.example.yml`](compose.example.yml) or create a `compose.yml`:

```yaml
services:
  mochy-navidrome-module:
    image: ghcr.io/1ofx9/mochy-navidrome-module:latest
    container_name: mochy-navidrome-module
    restart: unless-stopped
    ports:
      - "8090:8090"
    environment:
      - PORT=8090
      - LOG_LEVEL=info
      - LOG_FORMAT=text
      # Optional: set public URL if behind a reverse proxy
      # - PUBLIC_URL=https://music-module.example.com
      # Optional: pre-fill Navidrome server address (auto-hides Server URL in Mochy)
      # - NAVIDROME_URL=https://music.example.com
```

> **Tip**: If this container shares a Docker network with your Navidrome container, it will automatically discover `http://navidrome:4533` on startup. You do not need to configure `NAVIDROME_URL`.

Start the container:

```bash
docker compose up -d
```

### Option B: Docker Run

```bash
docker run -d \
  --name mochy-navidrome-module \
  --restart unless-stopped \
  -p 8090:8090 \
  -e LOG_LEVEL=info \
  ghcr.io/1ofx9/mochy-navidrome-module:latest
```

### Health Check

Verify that the module is healthy:

```bash
curl http://localhost:8090/health
# {"status":"ok","version":"1.4.1"}
```

---

## Reverse Proxy Setup (Optional)

If exposing the module through a reverse proxy, pass the incoming host and headers:

#### Caddy
```caddyfile
music-module.example.com {
    reverse_proxy localhost:8090
}
```

#### Nginx
```nginx
server {
    server_name music-module.example.com;

    location / {
        proxy_pass http://127.0.0.1:8090;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Set `PUBLIC_URL` when using a reverse proxy:
```bash
PUBLIC_URL=https://music-module.example.com
```

---

## Setup in Mochy Music (Android)

1. Open **Mochy Music** on your Android phone.
2. Navigate to **Settings** → **Modules** → **Add a module**.
3. Select **Add from a link** and enter your module manifest URL:
   - **Local Network**: `http://<LAN_IP>:8090/manifest.json`
   - **Reverse Proxy**: `https://music-module.example.com/manifest.json`
4. Tap **Add**. Mochy will register the **Navidrome** module.
5. In the module settings:
   - **Server URL**: Enter your Navidrome server address *(this field is automatically hidden if `NAVIDROME_URL` is set or auto-discovered on Docker)*.
   - **Username**: Your Navidrome username.
   - **Password**: Your Navidrome password.
   - **Audio Quality**: Preferred streaming quality (`hires`, `lossless`, `high`, or `low`).
6. Search and stream tracks from your Navidrome library directly in Mochy Music.

---

## Configuration Reference

Refer to [`.env.example`](.env.example) for environment variable defaults:

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8090` | HTTP port the module listens on. |
| `PUBLIC_URL` | *(empty)* | Optional public base URL. Detected from the `Host` header if empty. |
| `NAVIDROME_URL` | *(empty)* | Optional default Navidrome URL. Hides the "Server URL" input in Mochy when set. |
| `MODULE_ID` | `mochy-navidrome` | Unique module identifier. Change this when running multiple instances. |
| `MODULE_NAME` | `Navidrome` | Display name shown in Mochy Music (e.g. `Navidrome (Family)`). |
| `LOG_LEVEL` | `info` | Log verbosity: `debug`, `info`, `warn`, `error`. |
| `LOG_FORMAT` | `text` | Log format: `text` or `json`. |

---

## Multi-Server Setup (Multiple Navidrome Instances)

You can serve multiple Navidrome servers on the same host into Mochy Music simultaneously by assigning each container a unique `MODULE_ID`, `MODULE_NAME`, and port:

```yaml
services:
  # Instance 1: Personal Library
  module-personal:
    image: ghcr.io/1ofx9/mochy-navidrome-module:latest
    container_name: mochy-module-personal
    restart: unless-stopped
    ports:
      - "8090:8090"
    environment:
      - PORT=8090
      - MODULE_ID=navidrome-personal
      - MODULE_NAME=Navidrome (Personal)
      - NAVIDROME_URL=http://navidrome-personal:4533
    networks:
      - homelab

  # Instance 2: Family Library
  module-family:
    image: ghcr.io/1ofx9/mochy-navidrome-module:latest
    container_name: mochy-module-family
    restart: unless-stopped
    ports:
      - "8091:8091"
    environment:
      - PORT=8091
      - MODULE_ID=navidrome-family
      - MODULE_NAME=Navidrome (Family)
      - NAVIDROME_URL=http://navidrome-family:4533
    networks:
      - homelab
```

Both instances will appear as separate music providers side-by-side in Mochy Music under **Play from**, allowing one-tap switching between libraries.

---

## Development & Local Build

Requirements: Go 1.24+

```bash
# Clone the repository
git clone https://github.com/1ofx9/mochy-navidrome-module.git
cd mochy-navidrome-module

# Run unit tests
go test -v ./...

# Start server locally
go run ./cmd/module
```

---

## Security & Architecture

- **Stateless**: Does not store credentials, session tokens, or music files on disk.
- **Client-Side Storage**: Navidrome credentials live on your device in the Mochy Music app and are passed per request.
- **Subsonic Token Auth**: Uses Subsonic's salted MD5 token authentication (`hex(md5(password + salt))`); raw passwords are never sent in query strings.
- **Private**: Does not scrobble or log listening activity.
- **Bounded Streams**: Upstream payloads and artwork responses are bounded with size limits to prevent unbounded memory usage.

---

## License

This project is licensed under the [MIT License](LICENSE).
