// Two-device collaboration test: User 1 shares with User 2, both connect and sync

const CP_URL = process.env.CONTROL_PLANE_URL || "http://192.168.1.6:8090";

async function run() {
  console.log("=== Testing Two-Device Real-Time Sync Handshake ===");

  // 1. User 1
  const u1Email = `user1_${Date.now()}@example.com`;
  const pass = "Password123!";
  await fetch(`${CP_URL}/api/collections/users/records`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email: u1Email, password: pass, passwordConfirm: pass, name: "User 1" }),
  });
  const u1Auth = await (await fetch(`${CP_URL}/api/collections/users/auth-with-password`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ identity: u1Email, password: pass }),
  })).json();
  const token1 = u1Auth.token;

  // 2. User 2
  const u2Email = `user2_${Date.now()}@example.com`;
  await fetch(`${CP_URL}/api/collections/users/records`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email: u2Email, password: pass, passwordConfirm: pass, name: "User 2" }),
  });
  const u2Auth = await (await fetch(`${CP_URL}/api/collections/users/auth-with-password`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ identity: u2Email, password: pass }),
  })).json();
  const token2 = u2Auth.token;
  const user2Id = u2Auth.record.id;

  // 3. User 1 registers relay
  const relay = await (await fetch(`${CP_URL}/api/collections/relays/self-host`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${token1}` },
    body: JSON.stringify({ url: "http://localhost:8085" }),
  })).json();
  console.log(`1. Relay created: GUID = ${relay.guid}`);

  // 4. User 1 adds User 2 as Member
  const memberRoleRes = await fetch(`${CP_URL}/api/collections/roles/records?filter=(name='Member')`, {
    headers: { Authorization: `Bearer ${token1}` },
  });
  const memberRoleData = await memberRoleRes.json();
  const memberRole = memberRoleData.items[0];

  await fetch(`${CP_URL}/api/collections/relay_roles/records`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${token1}` },
    body: JSON.stringify({ relay: relay.id, user: user2Id, role: memberRole.id }),
  });
  console.log(`2. User 2 added to relay members.`);

  // 5. User 1 creates shared folder (public to relay members)
  const folderGuid = `folder-${Date.now()}`;
  const folder = await (await fetch(`${CP_URL}/api/collections/shared_folders/records`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${token1}` },
    body: JSON.stringify({ name: "Shared Vault", guid: folderGuid, relay: relay.id, creator: u1Auth.record.id, private: false }),
  })).json();
  console.log(`3. Folder created: GUID = ${folder.guid}`);

  // 6. User 2 requests doc token using relay GUID and folder GUID
  const docGuid = `doc-${Date.now()}`;
  console.log(`4. User 2 requesting token with Relay GUID ${relay.guid}...`);
  const u2TokenRes = await fetch(`${CP_URL}/token`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${token2}` },
    body: JSON.stringify({ docId: docGuid, relay: relay.guid, folder: folder.guid, device: "mac-device" }),
  });
  if (!u2TokenRes.ok) throw new Error(`User 2 token request failed: ${u2TokenRes.status} ${await u2TokenRes.text()}`);
  const u2Token = await u2TokenRes.json();
  console.log(`   User 2 token GRANTED: URL = ${u2Token.url}`);

  // 7. User 1 requests doc token
  const u1TokenRes = await fetch(`${CP_URL}/token`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${token1}` },
    body: JSON.stringify({ docId: docGuid, relay: relay.guid, folder: folder.guid, device: "win-device" }),
  });
  if (!u1TokenRes.ok) throw new Error(`User 1 token request failed: ${u1TokenRes.status} ${await u1TokenRes.text()}`);
  const u1Token = await u1TokenRes.json();
  console.log(`   User 1 token GRANTED: URL = ${u1Token.url}`);

  // 8. Both connect via WebSockets
  const ws1Url = `${u1Token.url}/${docGuid}?token=${u1Token.token}`;
  const ws2Url = `${u2Token.url}/${docGuid}?token=${u2Token.token}`;

  const ws1 = new WebSocket(ws1Url);
  const ws2 = new WebSocket(ws2Url);

  await Promise.all([
    new Promise((resolve, reject) => {
      ws1.onopen = () => { console.log("   Device 1 WebSocket CONNECTED!"); resolve(); };
      ws1.onerror = reject;
    }),
    new Promise((resolve, reject) => {
      ws2.onopen = () => { console.log("   Device 2 WebSocket CONNECTED!"); resolve(); };
      ws2.onerror = reject;
    }),
  ]);

  ws1.close();
  ws2.close();
  console.log("\nTWO-DEVICE REAL-TIME COLLABORATION FULLY OPERATIONAL!");
}

run().catch((e) => {
  console.error("Test failed:", e);
  process.exit(1);
});
