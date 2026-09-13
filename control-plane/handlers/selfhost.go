package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/open-relay/control-plane/crypto"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

type SelfHostRelayRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// RegisterSelfHostHandler registers POST /api/collections/relays/self-host
func RegisterSelfHostHandler(router *echo.Echo, app core.App, km *crypto.KeyManager) {
	router.POST("/api/collections/relays/self-host", func(c echo.Context) error {
			authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
			if authRecord == nil {
				return apis.NewUnauthorizedError("Authentication required", nil)
			}

			var req SelfHostRelayRequest
			if err := c.Bind(&req); err != nil {
				return apis.NewBadRequestError("Invalid request body", err)
			}

			req.URL = strings.TrimRight(strings.TrimSpace(req.URL), "/")
			if req.URL == "" {
				return apis.NewBadRequestError("Missing server URL", nil)
			}
			if req.Name == "" {
				req.Name = "Self-Hosted Relay"
			}

			dao := app.Dao()

			// 1. Find or create provider
			providersCol, err := dao.FindCollectionByNameOrId("providers")
			if err != nil {
				return apis.NewBadRequestError("providers collection missing", err)
			}

			providerRec, err := dao.FindFirstRecordByFilter("providers", "url = {:url}", dbx.Params{
				"url": req.URL,
			})
			if providerRec == nil {
				providerRec = models.NewRecord(providersCol)
				providerRec.Set("name", req.Name)
				providerRec.Set("url", req.URL)
				providerRec.Set("selfHosted", true)
				providerRec.Set("publicKey", km.PublicKeyBase64)
				providerRec.Set("keyType", "EdDSA")
				providerRec.Set("keyId", km.KeyID)
				if err := dao.SaveRecord(providerRec); err != nil {
					return apis.NewBadRequestError(fmt.Sprintf("Failed to save provider: %v", err), nil)
				}
			}

			// 2. Create relay record
			relaysCol, err := dao.FindCollectionByNameOrId("relays")
			if err != nil {
				return apis.NewBadRequestError("relays collection missing", err)
			}

			relayRec := models.NewRecord(relaysCol)
			relayRec.Set("guid", uuid.New().String())
			relayRec.Set("name", req.Name)
			relayRec.Set("version", 1)
			relayRec.Set("userLimit", 0)
			relayRec.Set("provider", providerRec.Id)
			relayRec.Set("storage_quota", "quota_foss_0001")
			relayRec.Set("plan", "Community FOSS")
			if err := dao.SaveRecord(relayRec); err != nil {
				return apis.NewBadRequestError(fmt.Sprintf("Failed to create relay: %v", err), nil)
			}

			// 3. Assign user as Owner in relay_roles
			relayRolesCol, err := dao.FindCollectionByNameOrId("relay_roles")
			if err != nil {
				return apis.NewBadRequestError("relay_roles collection missing", err)
			}

			roleRec := models.NewRecord(relayRolesCol)
			roleRec.Set("user", authRecord.Id)
			roleRec.Set("relay", relayRec.Id)
			roleRec.Set("role", "role_owner_001")
			if err := dao.SaveRecord(roleRec); err != nil {
				return apis.NewBadRequestError(fmt.Sprintf("Failed to assign owner role: %v", err), nil)
			}

			// 4. Create default invitation for Member
			invitesCol, err := dao.FindCollectionByNameOrId("relay_invitations")
			if err == nil && invitesCol != nil {
				inviteRec := models.NewRecord(invitesCol)
				inviteRec.Set("relay", relayRec.Id)
				inviteRec.Set("role", "role_member_002")
				keyBytes := make([]byte, 16)
				_, _ = rand.Read(keyBytes)
				inviteRec.Set("key", hex.EncodeToString(keyBytes))
				inviteRec.Set("enabled", true)
				_ = dao.SaveRecord(inviteRec)
			}

			return c.JSON(http.StatusOK, relayRec)
		}, apis.ActivityLogger(app))
}
