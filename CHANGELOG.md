# Changelog

All notable changes to this project are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and versions follow [SemVer](https://semver.org/).

## [Unreleased]

## [1.2.2] - 2026-11-20

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
