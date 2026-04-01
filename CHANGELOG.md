# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [2026-04-01]

### Added
- IP persistence to server certificate generation and renewal scripts to ensure correct SAN configuration.
- Automated TLS certificate management with cron support and `.env` configuration for server and client.
- `.env.server.example` template for server configuration.
- Environment example and maintenance scripts in portable build package.

### Changed
- Clarified TLS setup instructions and environment configuration for source and release deployments.

## [2026-03-31]

### Added
- Core tunnel infrastructure, including gRPC service, client routing, and portable build scripts.
- License file.
- Changelog file.
