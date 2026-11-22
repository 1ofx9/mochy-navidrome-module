# Changelog

All notable changes to this project are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and versions follow [SemVer](https://semver.org/).

## [Unreleased]

## [1.5.1] - 2026-11-22

### Changed

- Removed the "Server URL" input field from the Mochy Music module manifest settings. Upstream server addresses are now managed exclusively on the module host (via `NAVIDROME_URL`, `SERVERS`, or Docker network auto-discovery), streamlining the in-app setup to only credentials (`Username`, `Password`, and `Audio Quality`).
- Hardened artwork caching and stream proxy routing to use server-configured upstream addresses when client queries omit `navidrome_url`.

## [1.5.0] - 2026-11-22

### Added

- Single-container multi-server path routing via `SERVERS` (or `INSTANCES`) environment variable (e.g. `SERVERS=music:Navidrome:https://music.example.com,dolby:Navidrome (Dolby):https://dolby.example.com`).
- Generates isolated manifests and endpoints per library (`/<slug>/manifest.json`, `/<slug>/search`, `/<slug>/stream/{id}`, etc.) from one lightweight container.
- Root index (`/`) and multi-instance health check (`/health`) listing all configured servers with copyable manifest URLs.
- Dynamic startup banner listing all configured server instances and their respective Mochy manifest links.

## [1.4.1] - 2026-11-21

### Added

- Added album cover art URLs to `/search` results across all common aliases (`coverUrl`, `albumCover`, `artworkUrl`, `artwork`, and `thumbnail`), resolving blank cover art placeholders in Mochy's search and module test screens.
- Added `logo` attribute to `/manifest.json` and a `/logo.png` route alias alongside `/icon.png` to fix icon rendering in Mochy's playback source selector sheet.
- Added comprehensive password field masking hints and attributes (`inputType: "password"`, `format: "password"`, `secure: true`, `secret: true`, `password: true`, `masked: true`) in `/manifest.json`.

## [1.4.0] - 2026-11-21

### Added

- Support for `MODULE_ID` (default: `mochy-navidrome`) and `MODULE_NAME` (default: `Navidrome`) environment variables to run multiple module instances serving different Navidrome servers simultaneously in Mochy Music.
- Dynamic startup banner and manifest generation reflecting custom module names.
- Multi-instance Docker Compose reference and documentation.

## [1.3.0] - 2026-11-20

### Added

- Embedded high-resolution official Navidrome logo served directly at `/icon.png` with 1-year cache headers, replacing the placeholder gray letter icon in Mochy.
- Added dynamic cover art resolution query (`?size=`) with a higher default of 800px for sharper art on AMOLED displays.
- Added customizable search count query (`?count=`) and increased default result limit from 10 to 20 tracks.

### Fixed

- Masked password input field with `type: "password"` in module settings so passwords are never shown in plaintext.

### Changed

- Reordered audio quality options to place `Hi-Res (24-bit)` at the top of the quality list.

## [1.2.1] - 2026-11-20

### Changed

- Polished audio quality badge formatting in stream responses (`Hi-Res Lossless`, `Lossless`, `High`, `Low`) for cleaner aesthetics in the player.
- Added `https://<your-domain>/manifest.json` placeholder in the startup banner when `PUBLIC_URL` is unset.

## [1.2.0] - 2026-11-20

### Added

- GitHub Actions status badges (CI, Docker Publish, Release, License) in README.
- `compose.example.yml` reference template for Docker Compose deployments with network auto-discovery guidance.
- Updated `.env.example` with full configuration reference for all supported environment variables.

### Changed

- Streamlined documentation and refined security architecture overview.
- Cleaned up comments across packages and verified zero dead code.

## [1.1.0] - 2026-11-20

### Fixed

- Fixed audio stream buffering and premature cutoffs by lifting the 15-second client timeout on long-lived audio streaming connections (`GET /audio/{id}`).

### Added

- Auto-discovery of Navidrome servers on the local/Docker network (`http://navidrome:4533`).
- Auto-hides the "Server URL" setting in Mochy Music when a Navidrome server is configured or auto-discovered, leaving only Username and Password.
- Support for `NAVIDROME_URL` environment variable to configure the default Navidrome server address.
- Active upstream Navidrome credential validation on `/health` when credentials are provided in query parameters.
- Cleanly aligned request logging columns with descriptive action tags (`[resolve stream]`, `[audio stream]`, `[seek range]`) and HTTP 206 Partial Content status labeling.
- Explicit link to MIT license in README.

## [1.0.0] - 2026-11-20

### Added

- Native Mochy Music Web Module integration declaring Song Source, Lyrics Provider, and Song Info Provider (`GET /manifest.json`).
- In-app credential authentication configured in Mochy Music mobile settings (`navidrome_url`, `navidrome_user`, `navidrome_password`, and `quality`).
- Fast Subsonic search integration querying Navidrome libraries (`GET /search`).
- Seamless audio stream resolution (`GET /stream/{id}`) and high-performance stream proxy (`GET /audio/{id}`) supporting HTTP `Range` and `206 Partial Content` seeking.
- Synchronized LRC lyrics extraction via Navidrome OpenSubsonic `getLyricsBySongId` with fallback to standard Subsonic plain lyrics (`GET /lyrics`).
- Track metadata provider for Mochy's native "About song" view (`GET /metadata`).
- Thread-safe in-memory LRU artwork cache with SHA-256 ETag generation and HTTP 304 revalidation (`GET /art/{id}`).
- Health check probe endpoint (`GET /health`) and CLI healthcheck flag (`-healthcheck`).
- Standalone zero-dependency Go implementation with zero external packages.
- Multi-arch Docker container build (`linux/amd64`, `linux/arm64`) on lightweight Alpine runtime.
