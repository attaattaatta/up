# Changelog

All notable changes to this project are documented in this file.

## [1.1.0] - 2026-04-14

This release reflects the current workspace state after `v1.0.1`.

### Added

- Command-line flags for version, help, bind address, and port selection.
- Response header `X-Server-Version` on HTTP responses.
- Safer upload validation for malformed multipart payloads, empty files, duplicate targets, and path traversal in filenames.
- Cleanup of partially written files when an upload fails.
- MIME-based download handling with a text heuristic to force large text files to download while allowing images, video, and PDFs to open inline.
- Automatic creation of the `uploads` directory during startup.
- GitHub Actions workflows for CI builds and tag-based release publishing.

### Changed

- Startup output now prints the server version, upload endpoint, and file URL pattern.
- Error responses now return more specific failure details for upload and filesystem problems.
- README was rewritten into a fuller product overview with features, usage examples, and deployment notes for Nginx, systemd, ShareX, and Docker.

### Fixed

- Unexpected positional CLI arguments are now rejected with a usage hint.
- Empty port values are validated before the server starts.

## [1.0.1] - 2025-03-25

### Changed

- Default server and client port changed from `8080` to `5555`.
- README and shell client examples were updated to use the new default port.
- Project documentation was polished for both Bash and PowerShell usage examples.

## [1.0.0] - 2025-03-24

### Added

- Initial Go HTTP upload/download server implementation.
- Random per-upload path generation for shareable file URLs.
- Basic request logging to console and `server.log`.
- Bash upload helper script in `client/up.sh`.
- Tagged first stable release of the project.
