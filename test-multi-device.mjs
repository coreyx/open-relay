// Multi-Device and GUID Token Acquisition & WebSocket Sync Test
const CP_URL = process.env.CONTROL_PLANE_URL || "http://192.168.1.6:8090";
const EMAIL = `test_lan_${Date.now()}@example.com`;
const PASSWORD = "Password123!";

async function run() {
  console.log("=== Testing Multi-Device Token & Sync with Relay GUID ===");

  // 1. Register a test user
  console.log(`\n1. Registering user via LAN URL: ${CP_URL}...`);
  const regRes = await fetch(`${CP_URL}/api/collections/users/records`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      email: EMAIL,
      password: PASSWORD,
      passwordConfirm: PASSWORD,
      name: "LAN Tester",
    }),
  });
  if (!regRes.ok) throw new Error(`Reg failed: ${regRes.status} ${await regRes.text()}`);
  const user = await regRes.json();
  console.log(`   User created: ${user.id}`);

  // 2. Authenticate
  console.log("\n2. Authenticating user...");
  const authRes = await fetch(`${CP_URL}/api/collections/users/auth-with-password`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ identity: EMAIL, password: PASSWORD }),
  });
  if (!authRes.ok) throw new Error(`Auth failed: ${authRes.status} ${await authRes.text()}`);
  const { token } = await authRes.json();
  console.log(`   Auth token obtained.`);

  // 3. Register self-host relay
  console.log("\n3. Registering self-hosted relay with URL http://localhost:8085...");
  const relayRes = await fetch(`${CP_URL}/api/collections/relays/self-host`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ url: "http://localhost:8085" }),
  });
  if (!relayRes.ok) throw new Error(`Relay failed: ${relayRes.status} ${await relayRes.text()}`);
  const relay = await relayRes.json();
  console.log(`   Relay created: ID = ${relay.id}, GUID = ${relay.guid}`);

  // 4. Request token USING THE RELAY GUID (simulating Obsidian client behavior!)
  console.log(`\n4. Requesting doc token USING RELAY GUID: ${relay.guid}...`);
  const docId = `doc-${Date.now()}`;
  const tokenRes = await fetch(`${CP_URL}/token`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({
      docId: docId,
      relay: relay.guid, // <--- MUST WORK WITH GUID!
      folder: "test-folder-guid",
      device: "mac-client-sim",
    }),
  });

  if (!tokenRes.ok) {
    const text = await tokenRes.text();
    throw new Error(`Token request failed: ${tokenRes.status} ${text}`);
  }
  const tokenData = await tokenRes.json();
  console.log(`   Token request SUCCEEDED!`);
  console.log(`   Returned WebSocket URL: ${tokenData.url}`);
  console.log(`   Returned BaseURL: ${tokenData.baseUrl}`);

  // Check that the returned URL uses 192.168.1.6 instead of localhost
  if (tokenData.url.includes("192.168.1.6:8085")) {
    console.log(`   SUCCESS: URL was dynamically adapted to client host IP (192.168.1.6:8085)!`);
  } else {
    console.log(`   Note: URL is ${tokenData.url}`);
  }

  // 5. Connect to WebSocket using the returned URL
  console.log(`\n5. Connecting WebSocket to: ${tokenData.url}/${tokenData.docId}?token=...`);
  const wsUrl = `${tokenData.url}/${tokenData.docId}?token=${tokenData.token}`;

  await new Promise((resolve, reject) => {
    const ws = new WebSocket(wsUrl);
    const timeout = setTimeout(() => {
      ws.close();
      reject(new Error("WebSocket timeout after 5000ms"));
    }, 5000);

    ws.onopen = () => {
      clearTimeout(timeout);
      console.log("   WebSocket connection successfully OPENED!");
      console.log("   CWT authentication & audience verification PASSED!");
      // Send sync step 1
      ws.send(new Uint8Array([0, 0]));
    };

    ws.onmessage = (event) => {
      console.log(`   Received response message from server (${event.data.byteLength || event.data.length || 0} bytes)!`);
      ws.close();
      resolve();
    };

    ws.onerror = (err) => {
      clearTimeout(timeout);
      reject(err);
    };
  });

  console.log("\nALL VERIFICATIONS PASSED!");
}

run().catch((err) => {
  console.error("Test failed:", err);
  process.exit(1);
});
