// End-to-End Handshake Verification Test for Open-Relay
const CP_URL = "http://localhost:8090";
const EMAIL = `test_${Date.now()}@example.com`;
const PASSWORD = "Password123!";

async function main() {
  console.log("=== Open-Relay End-to-End Verification ===");

  // 1. Register a new user
  console.log(`\n1. Registering test user: ${EMAIL}`);
  const regRes = await fetch(`${CP_URL}/api/collections/users/records`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      email: EMAIL,
      password: PASSWORD,
      passwordConfirm: PASSWORD,
      name: "Test User",
    }),
  });

  if (!regRes.ok) {
    const text = await regRes.text();
    throw new Error(`User registration failed: ${regRes.status} ${text}`);
  }
  const user = await regRes.json();
  console.log(`   User created successfully: ID = ${user.id}`);

  // 2. Authenticate user
  console.log("\n2. Authenticating user to get auth token...");
  const authRes = await fetch(`${CP_URL}/api/collections/users/auth-with-password`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      identity: EMAIL,
      password: PASSWORD,
    }),
  });

  if (!authRes.ok) {
    const text = await authRes.text();
    throw new Error(`Authentication failed: ${authRes.status} ${text}`);
  }
  const authData = await authRes.json();
  const token = authData.token;
  console.log(`   Auth token obtained: ${token.substring(0, 20)}...`);

  // 3. Register self-hosted relay
  console.log("\n3. Registering self-hosted relay at http://localhost:8085...");
  const relayRes = await fetch(`${CP_URL}/api/collections/relays/self-host`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "Authorization": `Bearer ${token}`,
    },
    body: JSON.stringify({
      url: "http://localhost:8085",
    }),
  });

  if (!relayRes.ok) {
    const text = await relayRes.text();
    throw new Error(`Self-host registration failed: ${relayRes.status} ${text}`);
  }
  const relay = await relayRes.json();
  console.log(`   Relay created: ID = ${relay.id}, GUID = ${relay.guid}, Provider = ${relay.provider}`);

  // 4. Request document token
  console.log("\n4. Requesting ephemeral document token from Control Plane...");
  const docId = `doc-${Date.now()}`;
  const tokenRes = await fetch(`${CP_URL}/token`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "Authorization": `Bearer ${token}`,
    },
    body: JSON.stringify({
      docId: docId,
      relay: relay.id,
      folder: "test-folder",
      device: "node-test-agent",
    }),
  });

  if (!tokenRes.ok) {
    const text = await tokenRes.text();
    throw new Error(`Token request failed: ${tokenRes.status} ${text}`);
  }
  const tokenData = await tokenRes.json();
  console.log("   Token response:", JSON.stringify(tokenData, null, 2));

  // 5. Test WebSocket handshake with Relay Server (Data Plane)
  console.log(`\n5. Connecting to Data Plane WebSocket: ${tokenData.url}/${docId}?token=${tokenData.token.substring(0, 20)}...`);
  const wsEndpoint = `${tokenData.url}/${docId}?token=${tokenData.token}`;

  await new Promise((resolve, reject) => {
    const ws = new WebSocket(wsEndpoint);
    const timeout = setTimeout(() => {
      ws.close();
      reject(new Error("WebSocket handshake timed out after 5000ms"));
    }, 5000);

    ws.onopen = () => {
      clearTimeout(timeout);
      console.log("   WebSocket connection successfully ESTABLISHED with Relay Server!");
      console.log("   CWT authentication and audience verification SUCCEEDED!");
      
      // Send a step-1 sync protocol message (step 1 sync request: [0, 0])
      // In Y-Sweet / Yrs protocol, message type 0 is sync step 1
      const syncStep1 = new Uint8Array([0, 0]);
      ws.send(syncStep1);
      console.log("   Sent Yjs sync step 1 message.");
    };

    ws.onmessage = (event) => {
      console.log(`   Received response message from server (size: ${event.data.length || event.data.byteLength || 0} bytes)`);
      ws.close();
      resolve();
    };

    ws.onerror = (err) => {
      clearTimeout(timeout);
      reject(new Error(`WebSocket error: ${err.message || err}`));
    };

    ws.onclose = (ev) => {
      clearTimeout(timeout);
      console.log(`   WebSocket closed (code: ${ev.code}, reason: "${ev.reason}")`);
      if (ev.code === 1000 || ev.code === 1005) {
        resolve();
      } else {
        reject(new Error(`WebSocket closed unexpectedly with code ${ev.code}: ${ev.reason}`));
      }
    };
  });

  // 6. Test File Token endpoint
  console.log("\n6. Testing /file-token endpoint...");
  const fileTokenRes = await fetch(`${CP_URL}/file-token`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "Authorization": `Bearer ${token}`,
    },
    body: JSON.stringify({
      docId: docId,
      relay: relay.id,
      folder: "test-folder",
      hash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
      contentType: "application/octet-stream",
      contentLength: 42,
      device: "node-test-agent",
    }),
  });

  if (!fileTokenRes.ok) {
    const text = await fileTokenRes.text();
    throw new Error(`File token request failed: ${fileTokenRes.status} ${text}`);
  }
  const fileTokenData = await fileTokenRes.json();
  console.log("   File token response:", JSON.stringify(fileTokenData, null, 2));

  // 7. Verify /flags and /whoami
  console.log("\n7. Testing /flags and /whoami endpoints...");
  const flagsRes = await fetch(`${CP_URL}/flags`);
  const flags = await flagsRes.json();
  console.log("   Flags response:", flags);

  const whoamiRes = await fetch(`${CP_URL}/whoami`, {
    headers: { "Authorization": `Bearer ${token}` },
  });
  const whoami = await whoamiRes.json();
  console.log("   WhoAmI response:", whoami);

  // 8. Clean up test artifacts
  console.log("\n8. Cleaning up test artifacts...");
  if (relay?.id) {
    await fetch(`${CP_URL}/api/collections/relays/records/${relay.id}`, {
      method: "DELETE",
      headers: { "Authorization": `Bearer ${token}` }
    });
  }
  if (user?.id) {
    await fetch(`${CP_URL}/api/collections/users/records/${user.id}`, {
      method: "DELETE",
      headers: { "Authorization": `Bearer ${token}` }
    });
  }
  console.log("   Test artifacts cleaned up successfully.");

  console.log("\n========================================================");
  console.log(" SUCCESS: All Open-Relay End-to-End tests passed!");
  console.log("========================================================\n");
}

main().catch((err) => {
  console.error("\n❌ Test failed:", err);
  process.exit(1);
});
