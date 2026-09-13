package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/open-relay/control-plane/crypto"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

// RegisterRelayHooks registers lifecycle hooks for the relays collection
func RegisterRelayHooks(app core.App, km *crypto.KeyManager) {
	app.OnRecordBeforeCreateRequest("relays").Add(func(e *core.RecordCreateEvent) error {
		dao := app.Dao()

		// 1. Ensure non-empty name
		name := strings.TrimSpace(e.Record.GetString("name"))
		if name == "" {
			e.Record.Set("name", "Untitled Relay Server")
		}

		// 2. Ensure valid GUID
		if strings.TrimSpace(e.Record.GetString("guid")) == "" {
			e.Record.Set("guid", uuid.New().String())
		}

		// 3. Ensure storage quota
		if e.Record.GetString("storage_quota") == "" {
			e.Record.Set("storage_quota", "quota_foss_0001")
		}

		// 4. Ensure plan
		if e.Record.GetString("plan") == "" {
			e.Record.Set("plan", "Community FOSS")
		}

		// 5. Ensure provider is attached
		if e.Record.GetString("provider") == "" {
			providerRec, _ := dao.FindFirstRecordByFilter("providers", "url ~ '8085'")
			if providerRec == nil {
				providerRec, _ = dao.FindFirstRecordByFilter("providers", "1=1")
			}
			if providerRec != nil {
				e.Record.Set("provider", providerRec.Id)
			}
		}

		// 6. Ensure version is at least 1
		if e.Record.GetInt("version") == 0 {
			e.Record.Set("version", 1)
		}

		return nil
	})

	app.OnRecordAfterCreateRequest("relays").Add(func(e *core.RecordCreateEvent) error {
		var authRecord *models.Record
		if e.HttpContext != nil {
			authRecord, _ = e.HttpContext.Get(apis.ContextAuthRecordKey).(*models.Record)
		}
		if authRecord == nil {
			return nil
		}

		dao := app.Dao()

		// 1. Assign creator as Owner in relay_roles
		relayRolesCol, err := dao.FindCollectionByNameOrId("relay_roles")
		if err == nil && relayRolesCol != nil {
			existing, _ := dao.FindFirstRecordByFilter("relay_roles", "user = {:user} && relay = {:relay}", dbx.Params{
				"user":  authRecord.Id,
				"relay": e.Record.Id,
			})
			if existing == nil {
				roleRec := models.NewRecord(relayRolesCol)
				roleRec.Set("user", authRecord.Id)
				roleRec.Set("relay", e.Record.Id)
				roleRec.Set("role", "role_owner_001")
				if err := dao.SaveRecord(roleRec); err != nil {
					log.Printf("[RelayHook] Failed to assign owner role: %v\n", err)
				}
			}
		}

		// 2. Create default Member invitation
		invitesCol, err := dao.FindCollectionByNameOrId("relay_invitations")
		if err == nil && invitesCol != nil {
			inviteRec := models.NewRecord(invitesCol)
			inviteRec.Set("relay", e.Record.Id)
			inviteRec.Set("role", "role_member_002")
			keyBytes := make([]byte, 16)
			_, _ = rand.Read(keyBytes)
			inviteRec.Set("key", hex.EncodeToString(keyBytes))
			inviteRec.Set("enabled", true)
			if err := dao.SaveRecord(inviteRec); err != nil {
				log.Printf("[RelayHook] Failed to create default invitation: %v\n", err)
			}
		}

		return nil
	})

	// Assign creator as Owner in shared_folder_roles upon folder creation
	app.OnRecordAfterCreateRequest("shared_folders").Add(func(e *core.RecordCreateEvent) error {
		var authRecord *models.Record
		if e.HttpContext != nil {
			authRecord, _ = e.HttpContext.Get(apis.ContextAuthRecordKey).(*models.Record)
		}
		if authRecord == nil {
			return nil
		}

		dao := app.Dao()
		sfRolesCol, err := dao.FindCollectionByNameOrId("shared_folder_roles")
		if err == nil && sfRolesCol != nil {
			existing, _ := dao.FindFirstRecordByFilter("shared_folder_roles", "user = {:user} && shared_folder = {:folder}", dbx.Params{
				"user":   authRecord.Id,
				"folder": e.Record.Id,
			})
			if existing == nil {
				roleRec := models.NewRecord(sfRolesCol)
				roleRec.Set("user", authRecord.Id)
				roleRec.Set("shared_folder", e.Record.Id)
				roleRec.Set("role", "role_owner_001")
				if err := dao.SaveRecord(roleRec); err != nil {
					log.Printf("[FolderHook] Failed to assign folder owner role: %v\n", err)
				}
			}
		}

		return nil
	})
}
