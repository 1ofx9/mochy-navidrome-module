# Mochy Navidrome Module (`mochy-navidrome-module`)

A lightweight, stateless web module connecting [Navidrome](https://www.navidrome.org/) (and OpenSubsonic / Subsonic compatible music servers) to [Mochy Music](https://realidkroo.github.io/Mochy-Music/) on Android.

---

## Features

- **Native Mochy Web Module**: Implements `/manifest.json`, `/search`, `/stream/{id}`, `/audio/{id}`, `/art/{id}`, `/lyrics`, and `/metadata` per Mochy's web module specification.
- **Client-Side Authentication**: Credentials (`navidrome_url`, `navidrome_user`, `navidrome_password`, and `quality`) are entered in the Mochy Music app on your phone and sent with requests.
- **100% Stateless**: No databases, no disk storage, and no background sync tasks.
- **Lossless & Hi-Res Streaming**: Automatically detects and passes audio format badges (`flac`, `alac`, `aac`, `opus`, `mp3`), sample rate, and bit depth.
- **HTTP Range Seeking**: Supports HTTP `Range` headers and `206 Partial Content` for responsive audio seeking in the player.
- **Synchronized LRC Lyrics**: Extracts timed lyrics from Navidrome via OpenSubsonic `getLyricsBySongId` with fallback to standard Subsonic plain text lyrics.
- **Cover Art Proxy & Cache**: In-memory LRU artwork cache with SHA-256 ETags and HTTP 304 revalidation.
- **Zero External Go Dependencies**: Built exclusively using the Go standard library.

---

## Docker Deployment

### Option A: Docker Compose (Recommended)

Create a `compose.yml` file:

```yaml
services:
  mochy-navidrome-module:
    image: ghcr.io/1ofx9/mochy-navidrome-module:latest
    # Or build locally:
    # build: .
    container_name: mochy-navidrome-module
    restart: unless-stopped
    ports:
      - "8090:8090"
    environment:
      - PORT=8090
      - LOG_LEVEL=info
      - LOG_FORMAT=text
      # If using a reverse proxy or public domain, set your public URL:
      # - PUBLIC_URL=https://music-module.example.com
```

Start the container:

```bash
docker compose up -d
```

### Option B: Docker CLI (`docker run`)

Using the pre-built GHCR image:

```bash
docker run -d \
  --name mochy-navidrome-module \
  --restart unless-stopped \
  -p 8090:8090 \
  -e LOG_LEVEL=info \
  ghcr.io/1ofx9/mochy-navidrome-module:latest
```

Or building from source:

```bash
git clone https://github.com/1ofx9/mochy-navidrome-module.git
cd mochy-navidrome-module
docker build -t mochy-navidrome-module:latest .
docker run -d --name mochy-navidrome-module --restart unless-stopped -p 8090:8090 mochy-navidrome-module:latest
```

### Check Container Health

Verify that the module is running:

```bash
docker ps
# Status will indicate (healthy)
curl http://localhost:8090/health
# {"status":"ok","version":"1.0.0"}
```

---

## Reverse Proxy Setup (Optional but Recommended)

If you expose the module over the internet with HTTPS, configure your reverse proxy:

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

When using a reverse proxy, set the `PUBLIC_URL` environment variable:
```bash
PUBLIC_URL=https://music-module.example.com
```

---

## Setup in Mochy Music (Android)

1. Open **Mochy Music** on your Android device.
2. Go to **Settings** → **Modules** → **Add a module**.
3. Choose **Add from a link** and paste your module's URL:
   - **Local Network**: `http://<HOST_LAN_IP>:8090/manifest.json` *(make sure your phone is on the same Wi-Fi)*
   - **Domain / Reverse Proxy**: `https://music-module.example.com/manifest.json`
4. Tap **Add**. Mochy will register **Navidrome**.
5. On the **Navidrome** module page in Mochy, toggle **Enable** and fill in your settings:
   - **Server URL**: Your Navidrome server address (e.g. `http://192.168.1.100:4533` or `https://music.example.com`)
   - **Username**: Your Navidrome username
   - **Password**: Your Navidrome password
   - **Audio Quality**: Preferred streaming quality (`lossless`, `hires`, `high`, or `low`)
6. Search for any song in your Navidrome library and stream directly in Mochy Music.

---

## Configuration Reference

All environment variables are optional with sensible defaults:

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8090` | HTTP listening port |
| `PUBLIC_URL` | *(empty)* | Optional public base URL (e.g. `https://music-module.example.com`). Autodetected from `Host` header if empty. |
| `NAVIDROME_URL` | *(empty)* | Optional default Navidrome server URL pre-filled in Mochy Music settings (e.g. `https://music.example.com`). |
| `LOG_LEVEL` | `info` | Log verbosity (`debug`, `info`, `warn`, `error`) |
| `LOG_FORMAT` | `text` | Log format (`text` or `json`) |

---

## Run Locally with Go

```bash
# Clone the repository
git clone https://github.com/1ofx9/mochy-navidrome-module.git
cd mochy-navidrome-module

# Run unit tests
go test -v ./...

# Start the module server
go run ./cmd/module
```

---

## Security & Privacy

- **100% Stateless**: No credentials, session tokens, or playback logs are stored on disk.
- **Client-Side Auth**: Server URLs and passwords remain securely on your mobile device in Mochy Music.
- **Subsonic MD5 Token Auth**: Upstream Subsonic authentication uses MD5 token auth (`hex(md5(password + salt))`) with a cryptographically secure random salt (`crypto/rand`) for each request. Plaintext passwords never travel to Navidrome in query strings.
- **Zero Scrobbling / Tracking**: Zero scrobble or now-playing reports are ever sent, keeping listening habits completely private.
- **Security Headers**: Standard defense-in-depth headers applied globally (`X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, and `Cross-Origin-Resource-Policy: cross-origin`).
- **Input & URL Sanitization**: `navidrome_url` parameters are strictly parsed to require valid `http` or `https` schemes with non-empty hosts, and track IDs are sanitized against CRLF control characters.
- **Memory & Resource Caps**: Upstream Subsonic and artwork responses are strictly capped via `io.LimitReader` to prevent memory exhaustion (OOM) attacks.

---

## License

This project is licensed under the [MIT License](LICENSE).
