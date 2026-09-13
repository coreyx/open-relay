package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

type AcceptInviteRequest struct {
	Key string `json:"key"`
}

type RotateKeyRequest struct {
	ID string `json:"id"`
}

// RegisterInviteHandlers registers invitation endpoints.
func RegisterInviteHandlers(router *echo.Echo, app core.App) {
	// POST /api/accept-invitation
	router.POST("/api/accept-invitation", func(c echo.Context) error {
			authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
			if authRecord == nil {
				return apis.NewUnauthorizedError("Authentication required", nil)
			}

			var req AcceptInviteRequest
			if err := c.Bind(&req); err != nil || req.Key == "" {
				return apis.NewBadRequestError("Missing or invalid invite key", err)
			}

			dao := app.Dao()

			// Find valid invitation
			invite, err := dao.FindFirstRecordByFilter("relay_invitations", "key = {:key} && enabled = true", dbx.Params{
				"key": req.Key,
			})
			if err != nil || invite == nil {
				return apis.NewNotFoundError("Invitation not found or expired", nil)
			}

			relayID := invite.GetString("relay")
			roleID := invite.GetString("role")

			// Upsert relay_roles
			relayRolesCol, err := dao.FindCollectionByNameOrId("relay_roles")
			if err != nil {
				return apis.NewBadRequestError("relay_roles collection missing", err)
			}

			existingRole, err := dao.FindFirstRecordByFilter("relay_roles", "user = {:user} && relay = {:relay}", dbx.Params{
				"user":  authRecord.Id,
				"relay": relayID,
			})

			if existingRole != nil {
				existingRole.Set("role", roleID)
				if err := dao.SaveRecord(existingRole); err != nil {
					return apis.NewBadRequestError("Failed to update membership role", err)
				}
			} else {
				newRoleRec := models.NewRecord(relayRolesCol)
				newRoleRec.Set("user", authRecord.Id)
				newRoleRec.Set("relay", relayID)
				newRoleRec.Set("role", roleID)
				if err := dao.SaveRecord(newRoleRec); err != nil {
					return apis.NewBadRequestError("Failed to grant membership role", err)
				}
			}

			// Return the joined relay record
			relayRec, err := dao.FindRecordById("relays", relayID)
			if err != nil {
				return apis.NewNotFoundError("Relay not found", nil)
			}

			return c.JSON(http.StatusOK, relayRec)
		}, apis.ActivityLogger(app))

	// POST /api/rotate-key
	router.POST("/api/rotate-key", func(c echo.Context) error {
		authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if authRecord == nil {
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		var req RotateKeyRequest
		if err := c.Bind(&req); err != nil || req.ID == "" {
			return apis.NewBadRequestError("Missing or invalid invitation id", err)
		}

		dao := app.Dao()

		invite, err := dao.FindRecordById("relay_invitations", req.ID)
		if err != nil || invite == nil {
			return apis.NewNotFoundError("Invitation not found", nil)
		}

		relayID := invite.GetString("relay")

		// Check caller is Owner
		userRole, err := dao.FindFirstRecordByFilter("relay_roles", "user = {:user} && relay = {:relay}", dbx.Params{
			"user":  authRecord.Id,
			"relay": relayID,
		})
		if err != nil || userRole == nil {
			return apis.NewForbiddenError("Forbidden: no access to this relay", nil)
		}

		roleRec, err := dao.FindRecordById("roles", userRole.GetString("role"))
		if err != nil || strings.ToLower(roleRec.GetString("name")) != "owner" {
			return apis.NewForbiddenError("Forbidden: only owners can rotate invitation keys", nil)
		}

		// Generate new key
		newKeyBytes := make([]byte, 16)
		if _, err := rand.Read(newKeyBytes); err != nil {
			return apis.NewBadRequestError("Failed to generate random key", err)
		}
		newKey := hex.EncodeToString(newKeyBytes)

		invite.Set("key", newKey)
		if err := dao.SaveRecord(invite); err != nil {
			return apis.NewBadRequestError("Failed to update invitation key", err)
		}

		return c.JSON(http.StatusOK, invite)
	}, apis.ActivityLogger(app))
}
