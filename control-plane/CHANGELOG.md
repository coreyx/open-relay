# Changelog

All notable changes to the Open-Relay Control Plane project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [1.0.0] - 2026-09-12

### Added
- **Core Architecture & Persistence**:
  - Implemented standalone Go 1.22 + PocketBase v0.22 service replacing the proprietary System3 Relay Control Plane.
  - Automated SQLite migrations (`migrations/schema.go`) provisioning 8 core collections:
    - `roles`: seeded with `Owner`, `Member`, `Reader`, and System3-compatible IDs (`2arnubkcv7jpce8`, `x6lllh2qsf9lxk6`).
    - `providers`: records for relay server hosts and Ed25519 public keys.
    - `storage_quotas`: unmetered 1TB default quota allocation with support for unmetered flags.
    - `relays`: parent collaborative contexts configured with unmetered collaborator limit (`userLimit = 0`).
    - `shared_folders`: synchronized vault directory contexts with privacy toggles.
    - `relay_roles`: relation table mapping users and roles to relay servers.
    - `shared_folder_roles`: folder-level access control and read/write overrides.
    - `relay_invitations`: cryptographic invitation records for peer onboarding.
- **Authentication & Permissions**:
  - Direct email and password registration and login via PocketBase's native `/api/collections/users/auth-with-password`.
  - Configured `@request.auth.id != ""` access rules across collections to enable multi-device collaboration while protecting unauthorized access.
- **Cryptographic Token Service (`crypto/`)**:
  - `crypto/keys.go`: Automated Ed25519 keypair generation, disk persistence (`pb_data/ed25519.key`), and environment variable overrides (`ED25519_PRIVATE_KEY`, `ED25519_KEY_ID`).
  - `crypto/cwt.go`: RFC 8392 CBOR Web Token (CWT) builder with COSE Sign1 (RFC 8152, algorithm `-8` EdDSA) and Tag 61 encapsulation.
  - Generates URL-safe base64url encoded tokens containing claims: `iss`, `sub`, `aud`, `exp`, `iat`, `scope` (`doc:<id>:rw`), and `channel` (relay GUID).
- **REST Endpoints (`handlers/`)**:
  - `POST /token`: Ephemeral document and folder token issuance.
    - Supports both PocketBase 15-character Record IDs and 36-character GUID lookups.
    - Resolves access via `relay_roles` using relational record IDs.
    - Implements dynamic host adaptation: if a provider is configured with `localhost` or `127.0.0.1`, remote clients connecting over LAN or VPN (e.g. `192.168.1.6:8090`) receive dynamically adapted WebSocket URLs pointing to the caller's reachable IP address on the provider's port (`8085`).
  - `POST /file-token`: Content-addressed attachment and binary file token generation with SHA-256 hash claims.
  - `POST /api/collections/relays/self-host`: Automatic provider upsert and owned relay creation.
  - `POST /api/accept-invitation`: Invitation code redemption with transactional member role assignment.
  - `POST /api/rotate-key`: Relay owner-authenticated invitation token rotation.
  - `GET /templates/relay.toml`: Generates pre-populated TOML server configuration with the control plane's active public key.
  - Compatibility handlers: `GET /flags`, `GET /whoami`, `GET /relay/:guid/check-host`.
- **Packaging & Deployment**:
  - Multi-stage Dockerfile producing a minimal, secure Alpine Linux container.

### Changed
- Standardized document token expiration default to 3600 seconds (1 hour).
- Relaxed owner permission checks in `checkRelayAccess` to inspect role name `"owner"` dynamically rather than relying solely on hardcoded identifiers.
