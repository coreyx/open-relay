package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/open-relay/control-plane/crypto"
	"github.com/open-relay/control-plane/migrations"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tests"
)

func TestControlPlaneFlow(t *testing.T) {
	testApp, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	defer testApp.Cleanup()

	// 1. Run migrations
	if err := migrations.EnsureSchema(testApp); err != nil {
		t.Fatalf("EnsureSchema failed: %v", err)
	}

	// Verify roles seeded
	for _, roleID := range []string{"role_owner_001", "role_member_002", "role_reader_003"} {
		rec, err := testApp.Dao().FindRecordById("roles", roleID)
		if err != nil || rec == nil {
			t.Fatalf("Role %s not found: %v", roleID, err)
		}
	}

	// Verify storage quota seeded
	quota, err := testApp.Dao().FindRecordById("storage_quotas", "quota_foss_0001")
	if err != nil || quota == nil {
		t.Fatalf("Default storage quota not found: %v", err)
	}
	if quota.GetBool("metered") {
		t.Errorf("Expected FOSS quota to be unmetered")
	}

	// 2. Initialize KeyManager
	km, err := crypto.InitKeyManager(testApp.DataDir())
	if err != nil {
		t.Fatalf("InitKeyManager failed: %v", err)
	}

	// 3. Register handlers
	echoRouter := echo.New()
	RegisterTokenHandlers(echoRouter, testApp, km)
	RegisterInviteHandlers(echoRouter, testApp)
	RegisterSelfHostHandler(echoRouter, testApp, km)
	RegisterTemplateHandler(echoRouter, testApp, km)

	// Create test user
	usersCol, err := testApp.Dao().FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("Find users collection failed: %v", err)
	}
	user := models.NewRecord(usersCol)
	user.SetUsername("alice")
	user.SetEmail("alice@example.com")
	user.SetPassword("password123")
	user.Set("name", "Alice")
	if err := testApp.Dao().SaveRecord(user); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Test template endpoint
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/templates/relay.toml", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	templateHandler := func(c echo.Context) error {
		toml := km.PublicKeyBase64
		return c.String(http.StatusOK, toml)
	}
	if err := templateHandler(c); err != nil {
		t.Fatalf("Template handler error: %v", err)
	}
	if !strings.Contains(rec.Body.String(), km.PublicKeyBase64) {
		t.Errorf("Expected template to contain public key %s", km.PublicKeyBase64)
	}

	// Test self-host relay creation
	relaysCol, err := testApp.Dao().FindCollectionByNameOrId("relays")
	if err != nil {
		t.Fatalf("Find relays collection failed: %v", err)
	}
	relayRec := models.NewRecord(relaysCol)
	relayRec.Set("guid", "test-relay-guid-001")
	relayRec.Set("name", "My Personal Relay")
	relayRec.Set("userLimit", 0)
	relayRec.Set("storage_quota", "quota_foss_0001")
	if err := testApp.Dao().SaveRecord(relayRec); err != nil {
		t.Fatalf("Failed to save relay: %v", err)
	}

	// Assign owner role in relay_roles
	rrCol, err := testApp.Dao().FindCollectionByNameOrId("relay_roles")
	if err != nil {
		t.Fatalf("Find relay_roles failed: %v", err)
	}
	rr := models.NewRecord(rrCol)
	rr.Set("user", user.Id)
	rr.Set("relay", relayRec.Id)
	rr.Set("role", "role_owner_001")
	if err := testApp.Dao().SaveRecord(rr); err != nil {
		t.Fatalf("Failed to save relay_roles: %v", err)
	}

	// Test checkRelayAccess
	isReadOnly, rRec, _, err := checkRelayAccess(testApp, user.Id, relayRec.Id, "")
	if err != nil {
		t.Fatalf("checkRelayAccess error: %v", err)
	}
	if isReadOnly {
		t.Errorf("Owner role should not be read-only")
	}
	if rRec.GetString("guid") != "test-relay-guid-001" {
		t.Errorf("Unexpected relay guid: %s", rRec.GetString("guid"))
	}

	// Test doc token generation through KeyManager
	token, err := km.SignDocToken("doc_main", user.Id, "http://localhost:8080", rRec.GetString("guid"), isReadOnly, 3600)
	if err != nil {
		t.Fatalf("SignDocToken error: %v", err)
	}
	if !strings.HasPrefix(token, km.KeyID+".") {
		t.Errorf("Token should have format key_id.token, got %s", token)
	}
}

func TestFlagsAndWhoAmI(t *testing.T) {
	testApp, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	defer testApp.Cleanup()

	usersCol, err := testApp.Dao().FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("Find users collection failed: %v", err)
	}
	user := models.NewRecord(usersCol)
	user.SetUsername("bob")
	user.SetEmail("bob@example.com")
	user.SetPassword("password123")
	user.Set("name", "Bob")
	_ = testApp.Dao().SaveRecord(user)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(apis.ContextAuthRecordKey, user)

	whoAmIHandler := func(c echo.Context) error {
		authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		return c.JSON(http.StatusOK, map[string]any{
			"id":    authRecord.Id,
			"email": authRecord.Email(),
			"name":  authRecord.GetString("name"),
		})
	}

	if err := whoAmIHandler(c); err != nil {
		t.Fatalf("whoAmIHandler error: %v", err)
	}

	var res map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}
	if res["email"] != "bob@example.com" {
		t.Errorf("Expected bob@example.com, got %v", res["email"])
	}
}
