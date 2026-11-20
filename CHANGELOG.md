# Changelog

All notable changes to this project are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and versions follow [SemVer](https://semver.org/).

Entries are written **one line per paragraph and bullet** (no hard-wrapping) so that a section pasted into GitHub release notes reflows to full width instead of breaking at the source wrap points.

## [Unreleased]

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
