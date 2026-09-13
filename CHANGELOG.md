# Changelog

All notable changes to the Open-Relay project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [1.0.0] - 2026-09-12

### Added
- **Open-Relay Control Plane (`control-plane`)**:
  - Implemented standalone Go + PocketBase v0.22 service replacing the proprietary System3 Relay Control Plane.
  - Automated database schema migrations and seed records across 8 core collections:
    - `roles` (`Owner`, `Member`, `Reader`, and System3-compatible IDs `2arnubkcv7jpce8`, `x6lllh2qsf9lxk6`)
    - `providers` (relay server hosts and public keys)
    - `storage_quotas` (default unmetered 1TB storage quota)
    - `relays` (shared folder parent contexts with unmetered collaborator limit `userLimit = 0`)
    - `shared_folders` (vault folder sync entities with public/private access modes)
    - `relay_roles` (access control mapping users to relays)
    - `shared_folder_roles` (folder-level read/write overrides)
    - `relay_invitations` (cryptographic invitation tokens for peer onboarding)
  - Multi-device adaptive host routing in `getProviderURL`:
    - Dynamically detects incoming `Host` headers from LAN (`192.168.x.x`) or VPN (`100.x.x.x`) callers.
    - Rewrites loopback `localhost:8085` provider URLs to the caller's reachable network address so secondary machines can connect to the relay data plane.
  - Robust identifier resolution in `/token`:
    - Accepts both 15-character PocketBase Record IDs and 36-character GUIDs for relays and folders.
    - Resolves access relations via `relayRec.Id` to maintain database foreign key integrity.
  - Cryptographic key manager (`crypto/keys.go`, `crypto/cwt.go`):
    - Ed25519 keypair generation, disk persistence, and environment variable loading (`ED25519_PRIVATE_KEY`, `ED25519_KEY_ID`).
    - RFC 8392 CBOR Web Token (CWT) builder with COSE Sign1 (RFC 8152, algorithm `-8` EdDSA) and Tag 61 wrapping.
    - Pure base64url serialization matching the relay data plane's native token detector.
  - Direct REST API endpoints:
    - `POST /token`: membership-validated ephemeral CWT document token issuance (`rw` / `r`).
    - `POST /file-token`: attachment and binary asset upload/download token generation.
    - `POST /api/collections/relays/self-host`: automatic provider upsert and owned relay registration.
    - `POST /api/accept-invitation`: invitation key redemption with automated role assignment.
    - `POST /api/rotate-key`: owner-authenticated invitation token rotation.
    - `GET /templates/relay.toml`: dynamic configuration generator pre-populated with active public keys.
    - `GET /flags`, `GET /whoami`, `GET /relay/:guid/check-host`: compatibility endpoints for the client plugin.
- **Relay Server Data Plane (`relay-server`)**:
  - Flexible audience validation (`crates/y-sweet-core/src/cwt.rs`):
    - Validates CWT tokens across `localhost`, LAN IPs, and Tailscale VPN endpoints by port matching (`:8085`) and wildcard (`*`) support, eliminating spurious cross-device audience rejections.
  - Multi-stage Docker build producing an optimized Debian-based container with Tailscale support.
  - Server configuration (`relay.toml`) with Ed25519 public key authentication (`openrelay_2026`).
  - Host port mapping configured to `8085` to avoid port contention with existing local HTTP services.
- **Obsidian Client Plugin (`Relay`)**:
  - Unlocked endpoint configuration allowing both `http:` and `https:` connection URLs across all environments.
  - Bypassed proprietary RSA JWT license check (`ENDPOINT_VALIDATION_PUBLIC_KEY` check against `api.system3.md`).
  - Added direct email and password authentication methods (`loginWithPassword`, `registerWithPassword`) in `LoginManager.ts`.
  - Added an inline email/password login and user registration interface in `LoggedIn.svelte`.
  - Added a "⚙️ Configure Relay Server" shortcut button for direct endpoint configuration.
  - Defensive entity lookups in `RelayManager.ts` preventing `Unable to find user` and `invalid role` crashes during real-time synchronization.
  - Added "Make public" button in `ManageRemoteFolder.svelte` to allow owners to toggle shared folder visibility.
  - Immediate connection on vault attachment in `Relays.svelte` (`folder.connect()` and `notifyListeners()`).
  - Built production plugin bundle in `Relay/main.js`.
- **Tooling & Orchestration**:
  - Root `compose.yaml` for 1-command startup of `open-relay-control` and `relay-server`.
  - Multi-root VS Code workspace (`open-relay.code-workspace`) linking all 5 workspace components.
  - Automated test suites:
    - `test-e2e.mjs`: single-user end-to-end registration, self-host setup, and CRDT handshake.
    - `test-multi-device.mjs`: LAN IP routing and Relay GUID token generation.
    - `test-two-devices.mjs`: two-device real-time sync with peer invitation and concurrent WebSocket sessions.
  - Individual `CHANGELOG.md` and `RELEASE_NOTES.md` documents across all multi-root workspace projects.

### Changed
- Configured default Relay data plane audience to `http://localhost:8085`.
- Updated `crates/run.sh` in the relay server to normalize CRLF line endings automatically during container build.
- Updated `tasks.md` tracking full completion across all 9 technical implementation milestones.

### Security
- Completely eliminated proprietary phone-home licensing checks and telemetry tracking.
- All tokens are verified locally using RFC 8392 / RFC 8152 Ed25519 signatures and audience matching.
- Removed cloud dependency on `api.system3.md` and `auth.system3.md`.
