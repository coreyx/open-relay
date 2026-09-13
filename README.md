# Open-Relay

**Open-Relay** is a 100% free and open-source (FOSS) self-hosted backend and client for real-time collaboration in Obsidian, replacing the proprietary "Relay Control Plane" with an unmetered, self-hosted architecture based on [PocketBase](https://pocketbase.io/) and [Yrs/Axum](https://github.com/No-Instructions/relay-server).

---

## Features

- **100% Free & Open Source**: Zero commercial licensing checks, zero collaborator limits, and no phone-home telemetry.
- **Complete Self-Hosting**: One-command deployment for both the Control Plane and Relay Data Plane via Docker Compose.
- **PocketBase Control Plane**: Lightweight, single-binary Go service backed by SQLite + WAL mode.
- **Cryptographic Security**: RFC 8392 CBOR Web Tokens (CWT) signed via COSE Sign1 (RFC 8152 EdDSA Ed25519).
- **Obsidian Plugin Compatibility**: Direct support for the Obsidian Relay plugin without requiring proprietary licenses or enterprise subscriptions.
- **Flexible Authentication**: Supports both direct email/password accounts and OAuth2 providers (Google, GitHub, Microsoft, Discord, OIDC).

---

## Architecture

```
                      ┌─────────────────────────────────────────┐
                      │             Obsidian Client             │
                      │         (Open-Relay Plugin)             │
                      └──────────────┬──────────────────┬───────┘
                                     │                  │
                1. REST Auth & Token │                  │ 3. Real-Time Yjs Sync
                   (HTTP/JSON)       │                  │    (WebSocket / CWT)
                                     ▼                  ▼
              ┌───────────────────────────┐    ┌───────────────────────────┐
              │    Open-Relay Control     │    │       Relay Server        │
              │  (Go / PocketBase :8090)  │    │   (Rust / Yrs / Axum :8080)│
              └──────────────┬────────────┘    └───────────────────────────┘
                             │                               ▲
                             │ 2. Issues Signed CWT Token    │
                             │    (Ed25519 / Tag 18 / Tag 61)│
                             └───────────────────────────────┘
```

1. **Control Plane (`control-plane`)**:
   - Manages vaults, users, relays, invitations, and permissions.
   - Signs ephemeral CBOR Web Tokens (CWT) using Ed25519.
   - Built with Go and PocketBase v0.22.
2. **Data Plane (`relay-server`)**:
   - Real-time CRDT sync server implementing Yjs document synchronization and awareness over WebSockets.
   - Authenticates clients via CWT tokens against its `relay.toml` public key.
3. **Client (`Relay`)**:
   - Obsidian community plugin for live peer-to-peer editing, presence, and document versioning.

---

## Quick Start (Docker Compose)

### 1. Clone the repository
```bash
git clone https://github.com/coreyx/open-relay.git
cd open-relay
```

### 2. Start the services
```bash
docker compose up -d
```

This starts:
- **Control Plane**: `http://localhost:8090` (Admin UI: `http://localhost:8090/_/`)
- **Relay Server**: `http://localhost:8080` (WebSocket: `ws://localhost:8080`)

### 3. Create an initial account
Open `http://localhost:8090/_/` in your browser to create your admin account, or create a user account directly in the Obsidian plugin.

---

## Installing the Obsidian Plugin

1. Build the plugin (or use pre-built artifacts in `Relay/`):
   ```bash
   cd Relay
   npm install
   npm run build
   ```
2. Copy the following files to your Obsidian vault's plugin directory:
   `<Vault>/.obsidian/plugins/system3-relay/`
   - `main.js`
   - `manifest.json`
   - `styles.css`
3. Reload Obsidian and enable the **Relay** plugin in Settings $\rightarrow$ Community Plugins.
4. In Relay settings, click **⚙️ Configure Relay Server** (or run the command `Configure self-hosted server / endpoints`).
5. Enter your Control Plane URL (e.g. `http://localhost:8090` or `https://relay.yourdomain.com`).
6. Register or log in with your email and password!

---

## Configuration

### Control Plane (`control-plane`)
Environment variables:
- `ED25519_KEY_ID`: Identifier for the signing key (default: `openrelay_2026`).
- `ED25519_PRIVATE_KEY`: Base64-encoded 32-byte seed or 64-byte Ed25519 private key.
- `RELAY_SERVER_URL`: Default URL of the data plane (e.g. `http://localhost:8080` or `http://relay-server:8080`).

### Relay Server (`relay.toml`)
```toml
[server]
host = "0.0.0.0"
port = 8080
url = "http://localhost:8085"

[store]
type = "filesystem"
path = "/app/data"

[[auth]]
key_id = "openrelay_2026"
public_key = "<BASE64_ED25519_PUBLIC_KEY>"
```

You can also download a pre-filled `relay.toml` anytime from `http://localhost:8090/templates/relay.toml`.

---

## Verification & Testing

Open-Relay includes automated end-to-end and multi-device integration test suites:

```bash
# 1. Single-user E2E registration, self-host setup, and CRDT handshake
node test-e2e.mjs

# 2. Multi-device LAN IP routing and Relay GUID token generation
node test-multi-device.mjs

# 3. Two-device real-time sync with peer invitation and concurrent WebSockets
node test-two-devices.mjs
```

---

## License

This project is licensed under the MIT and Apache-2.0 licenses.
