# Release Notes: Open-Relay v1.0.0 (Initial FOSS Release)

**Release Date:** September 12, 2026  
**License:** MIT / Apache-2.0  

---

## Overview

We are thrilled to introduce **Open-Relay v1.0.0**, a 100% free and open-source (FOSS) self-hosted collaboration platform for the Obsidian [Relay plugin](https://community.obsidian.md/plugins/system3-relay).

Until now, using Relay required connecting to the proprietary "Relay Control Plane" (`system3.md`), subject to commercial license validation, restrictive collaborator caps, metered storage quotas, and proprietary telemetry. 

Open-Relay completely replaces the proprietary control plane with a lightweight Go + PocketBase service, removes all client-side gatekeeping, and containerizes the real-time CRDT sync server for effortless self-hosting across local networks, VPNs, and the internet.

---

## Key Highlights

### 1. 100% Free & Open Source
- **Zero Collaborator Limits:** Share vaults and folders with unlimited team members, friends, or family (`userLimit = 0`).
- **Unmetered Quotas:** Uncapped folder storage quotas with support for unmetered plans.
- **Zero Proprietary Dependencies:** No connection to `api.system3.md`, `auth.system3.md`, or external license servers.

### 2. Complete Self-Hosting in One Command
A single `docker compose up -d` launches both the Control Plane and Relay Server:
- **Control Plane (`open-relay-control`)**: Port `8090` (REST API & PocketBase Admin UI).
- **Relay Server (`relay-server`)**: Port `8085` (WebSocket real-time Yrs CRDT sync).

### 3. Native Email & Password Authentication
- Users can register and log in directly within the Obsidian plugin settings using standard email and password credentials.
- Optional OAuth2 providers (GitHub, Google, Microsoft, Discord, OIDC) are supported through PocketBase.

### 4. Multi-Device LAN & VPN Synchronization
- **Adaptive Host Routing**: Automatically adapts local server addresses so remote laptops, desktops, and mobile devices on your LAN (`192.168.x.x`) or Tailscale network (`100.x.x.x`) can connect seamlessly.
- **Flexible Audience Verification**: Data plane accepts validly signed Ed25519 tokens across any local network interface matching the relay service port (`:8085`).
- **Resilient Entity Models**: Defensive getters in the client prevent crashes when remote peers join relays in real time.

### 5. Cryptographic RFC 8392 CWT & COSE Sign1
- Ephemeral tokens are minted using RFC 8392 CBOR Web Tokens (CWT) signed with Ed25519 (RFC 8152 COSE Sign1, algorithm `-8`).
- Role-based permissions (`Owner`, `Member`, `Reader`) ensure granular read-only vs read-write access.

### 6. Multi-Root Developer Experience
- Included `open-relay.code-workspace` allows seamless development across Go, Rust, and TypeScript/Svelte with unified workspace settings and language server recommendations.
- Dedicated `CHANGELOG.md` and `RELEASE_NOTES.md` documents across all 5 workspace projects.

---

## Workspace Projects Directory

| Workspace Folder | Path | Technology | Documentation |
|---|---|---|---|
| **Open-Relay (Root)** | `.` | Docker Compose, Node.js | [CHANGELOG](file:///c:/Users/corey/source/repos/open-relay/CHANGELOG.md) / [RELEASE_NOTES](file:///c:/Users/corey/source/repos/open-relay/RELEASE_NOTES.md) |
| **Control Plane** | `control-plane/` | Go 1.22, PocketBase v0.22 | [CHANGELOG](file:///c:/Users/corey/source/repos/open-relay/control-plane/CHANGELOG.md) / [RELEASE_NOTES](file:///c:/Users/corey/source/repos/open-relay/control-plane/RELEASE_NOTES.md) |
| **Relay Server** | `relay-server/` | Rust 1.89, Axum, Yrs | [CHANGELOG](file:///c:/Users/corey/source/repos/open-relay/relay-server/CHANGELOG.md) / [RELEASE_NOTES](file:///c:/Users/corey/source/repos/open-relay/relay-server/RELEASE_NOTES.md) |
| **Relay Client** | `Relay/` | TypeScript 5.4, Svelte 5 | [CHANGELOG](file:///c:/Users/corey/source/repos/open-relay/Relay/CHANGELOG.md) / [RELEASE_NOTES](file:///c:/Users/corey/source/repos/open-relay/Relay/RELEASE_NOTES.md) |
| **Specifications** | `spec/` | Markdown (EARS, Architecture) | [CHANGELOG](file:///c:/Users/corey/source/repos/open-relay/spec/CHANGELOG.md) / [RELEASE_NOTES](file:///c:/Users/corey/source/repos/open-relay/spec/RELEASE_NOTES.md) |

---

## Quickstart Guide

### 1. Run the Backend
```bash
git clone https://github.com/coreyx/open-relay.git
cd open-relay
docker compose up -d
```

Verify services:
- PocketBase Admin Console: `http://localhost:8090/_/`
- Relay Server WebSocket: `ws://localhost:8085`

### 2. Install the Obsidian Plugin
1. Open your Obsidian vault directory.
2. Create the plugin folder:
   ```text
   <Vault>/.obsidian/plugins/open-relay-client/
   ```
3. Copy `main.js`, `manifest.json`, and `styles.css` from `open-relay/Relay/` into that directory.
4. Reload Obsidian and toggle **Relay** on in **Community Plugins**.

### 3. Connect to Open-Relay
1. In Obsidian, open **Settings $\rightarrow$ Relay**.
2. Click **⚙️ Configure Relay Server** and set your Control Plane URL:
   - On the host machine: `http://localhost:8090`
   - On a secondary device (laptop/mobile): `http://<HOST_IP>:8090` (e.g. `http://192.168.1.6:8090`)
3. Enter your email and password, click **Sign Up** (or **Log In**).
4. Create or join a shared folder and start collaborating!

---

## Automated Test Verification

Open-Relay includes end-to-end integration tests validating multi-device networking and sync:
- `test-e2e.mjs`: Complete single-device authentication, relay provisioning, and CWT handshake $\rightarrow$ **PASS**
- `test-multi-device.mjs`: LAN IP routing and Relay GUID token generation $\rightarrow$ **PASS**
- `test-two-devices.mjs`: Two distinct users across LAN endpoints performing real-time CRDT WebSocket sync $\rightarrow$ **PASS**
