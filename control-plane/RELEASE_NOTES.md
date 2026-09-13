# Release Notes: Open-Relay Control Plane v1.0.0

**Release Date:** September 12, 2026  
**Module:** `github.com/open-relay/control-plane`  
**Runtime:** Go 1.22 / PocketBase v0.22  
**License:** MIT  

---

## Overview

The **Open-Relay Control Plane** is a lightweight, self-hosted replacement for the proprietary System3 Relay Control Plane. It manages user authentication, relational vault metadata, role-based access control, cryptographic Ed25519 CWT token minting, and relay peer invitations.

By deploying the Control Plane, users can host their own collaborative Obsidian syncing infrastructure with zero external cloud dependencies, unmetered storage limits, and unlimited collaboration seats.

---

## Key Features

### 1. Embedded PocketBase Framework
- Built directly on PocketBase v0.22, leveraging embedded SQLite with write-ahead logging (WAL) for ultra-fast, zero-maintenance data persistence.
- Provides a clean administrative dashboard accessible at `http://localhost:8090/_/` for inspecting users, relays, and access records.
- Automated migrations initialize all required database schemas and default roles upon container startup.

### 2. Native Email/Password Authentication
- Enables direct user registration and password-based login through standard REST endpoints.
- Avoids forced third-party SSO or commercial authorization proxies while maintaining full compatibility with optional OAuth2 providers.

### 3. Cryptographic Token Minting (RFC 8392 CWT & COSE Sign1)
- Eliminates cloud token dependencies by signing all client access tokens locally using Ed25519 (algorithm `-8`).
- Ephemeral tokens feature standard RFC 8392 CBOR Web Token (CWT) formatting and COSE Sign1 encapsulation.
- Signs specific document permissions (`doc:<id>:rw` for writers, `doc:<id>:r` for readers) and ties them to relay channels for strict isolation.

### 4. Adaptive Multi-Device LAN Routing
- Automatically detects the incoming `Host` header from remote client requests (e.g. `192.168.1.6:8090` or VPN addresses).
- When a provider is configured with loopback addresses (`localhost`), the control plane dynamically rewrites the returned WebSocket URL to the caller's reachable address, ensuring secondary laptops, desktops, and mobile devices connect seamlessly to the Relay Server.

### 5. Flexible Identifier Resolution
- Seamlessly resolves relays and shared folders by either PocketBase internal record IDs (15 characters) or global unique identifiers (GUIDs, 36 characters) used by the Obsidian client.

---

## Configuration & Environment Variables

| Variable | Default | Description |
|---|---|---|
| `ED25519_KEY_ID` | `openrelay_2026` | Key identifier attached to the COSE protected header (`kid`). |
| `ED25519_PRIVATE_KEY` | *(Generated on boot)* | Base64-encoded 32-byte seed or 64-byte Ed25519 private key. |
| `RELAY_SERVER_URL` | `http://relay-server:8080` | Internal container network address of the Relay Server data plane. |

---

## Endpoints Reference

- `POST /token`: Document and folder ephemeral CWT token generation.
- `POST /file-token`: Ephemeral token for binary file and attachment uploads.
- `POST /api/collections/relays/self-host`: Registers a self-hosted relay and associates provider configuration.
- `POST /api/accept-invitation`: Redeems a cryptographic invitation token and assigns member roles.
- `POST /api/rotate-key`: Rotates the shared invitation key for a relay.
- `GET /templates/relay.toml`: Pre-fills a `relay.toml` file with the server's public signing key.
- `GET /flags`, `GET /whoami`, `GET /relay/:guid/check-host`: Client compatibility endpoints.
