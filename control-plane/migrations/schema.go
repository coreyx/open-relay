package migrations

import (
	"log"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
	"github.com/pocketbase/pocketbase/tools/types"
)

// EnsureSchema initializes all collections, fields, indexes, and seed records.
func EnsureSchema(app core.App) error {
	dao := app.Dao()

	usersCol, err := dao.FindCollectionByNameOrId("users")
	if err != nil {
		return err
	}
	authRule := types.Pointer("@request.auth.id != ''")
	usersCol.ViewRule = authRule
	usersCol.ListRule = authRule
	if err := dao.SaveCollection(usersCol); err != nil {
		log.Printf("[Migration] Warning updating users rules: %v\n", err)
	}

	// 1. roles collection
	rolesCol, err := dao.FindCollectionByNameOrId("roles")
	if err != nil {
		log.Println("[Migration] Creating 'roles' collection...")
		allRule := types.Pointer("@request.auth.id != ''")
		rolesCol = &models.Collection{
			Name:       "roles",
			Type:       models.CollectionTypeBase,
			ListRule:   allRule,
			ViewRule:   allRule,
			CreateRule: nil, // Only system/admin
			UpdateRule: nil,
			DeleteRule: nil,
			Schema: schema.NewSchema(
				&schema.SchemaField{
					Name:     "name",
					Type:     schema.FieldTypeText,
					Required: true,
				},
			),
		}
		if err := dao.SaveCollection(rolesCol); err != nil {
			return err
		}
	}

	// Seed roles
	roles := []struct {
		ID   string
		Name string
	}{
		{ID: "role_owner_001", Name: "Owner"},
		{ID: "role_member_002", Name: "Member"},
		{ID: "role_reader_003", Name: "Reader"},
		{ID: "2arnubkcv7jpce8", Name: "Owner"},
		{ID: "x6lllh2qsf9lxk6", Name: "Member"},
	}
	for _, r := range roles {
		if record, _ := dao.FindRecordById("roles", r.ID); record == nil {
			rec := models.NewRecord(rolesCol)
			rec.SetId(r.ID)
			rec.Set("name", r.Name)
			if err := dao.SaveRecord(rec); err != nil {
				log.Printf("[Migration] Error seeding role %s: %v\n", r.Name, err)
			}
		}
	}

	// 2. providers collection
	providersCol, err := dao.FindCollectionByNameOrId("providers")
	if err != nil {
		log.Println("[Migration] Creating 'providers' collection...")
		allRule := types.Pointer("@request.auth.id != ''")
		providersCol = &models.Collection{
			Name:       "providers",
			Type:       models.CollectionTypeBase,
			ListRule:   allRule,
			ViewRule:   allRule,
			CreateRule: allRule,
			UpdateRule: allRule,
			DeleteRule: allRule,
			Schema: schema.NewSchema(
				&schema.SchemaField{
					Name:     "name",
					Type:     schema.FieldTypeText,
					Required: true,
				},
				&schema.SchemaField{
					Name:     "url",
					Type:     schema.FieldTypeText,
					Required: true,
				},
				&schema.SchemaField{
					Name: "selfHosted",
					Type: schema.FieldTypeBool,
				},
				&schema.SchemaField{
					Name: "publicKey",
					Type: schema.FieldTypeText,
				},
				&schema.SchemaField{
					Name: "keyType",
					Type: schema.FieldTypeText,
				},
				&schema.SchemaField{
					Name: "keyId",
					Type: schema.FieldTypeText,
				},
			),
		}
		if err := dao.SaveCollection(providersCol); err != nil {
			return err
		}
	}

	// Seed default provider if none exists
	if count, _ := dao.FindRecordsByFilter("providers", "1=1", "", 1, 0); len(count) == 0 {
		rec := models.NewRecord(providersCol)
		rec.SetId("provider_foss_001")
		rec.Set("name", "Self-Hosted Relay")
		rec.Set("url", "http://localhost:8085")
		rec.Set("selfHosted", true)
		rec.Set("keyType", "EdDSA")
		_ = dao.SaveRecord(rec)
	}

	// Ensure any providers pointing to 8080 are migrated to 8085
	p8080, _ := dao.FindRecordsByFilter("providers", "url ~ '8080'", "", 0, 0)
	for _, p := range p8080 {
		p.Set("url", "http://localhost:8085")
		_ = dao.SaveRecord(p)
	}

	// 3. storage_quotas collection
	quotasCol, err := dao.FindCollectionByNameOrId("storage_quotas")
	if err != nil {
		log.Println("[Migration] Creating 'storage_quotas' collection...")
		allRule := types.Pointer("@request.auth.id != ''")
		quotasCol = &models.Collection{
			Name:       "storage_quotas",
			Type:       models.CollectionTypeBase,
			ListRule:   allRule,
			ViewRule:   allRule,
			CreateRule: nil,
			UpdateRule: nil,
			DeleteRule: nil,
			Schema: schema.NewSchema(
				&schema.SchemaField{
					Name:     "name",
					Type:     schema.FieldTypeText,
					Required: true,
				},
				&schema.SchemaField{
					Name: "quota",
					Type: schema.FieldTypeNumber,
				},
				&schema.SchemaField{
					Name: "usage",
					Type: schema.FieldTypeNumber,
				},
				&schema.SchemaField{
					Name: "maxFileSize",
					Type: schema.FieldTypeNumber,
				},
				&schema.SchemaField{
					Name: "max_file_size",
					Type: schema.FieldTypeNumber,
				},
				&schema.SchemaField{
					Name: "metered",
					Type: schema.FieldTypeBool,
				},
			),
		}
		if err := dao.SaveCollection(quotasCol); err != nil {
			return err
		}
	} else {
		if f := quotasCol.Schema.GetFieldByName("max_file_size"); f == nil {
			quotasCol.Schema.AddField(&schema.SchemaField{
				Name: "max_file_size",
				Type: schema.FieldTypeNumber,
			})
			_ = dao.SaveCollection(quotasCol)
		}
	}

	// Seed default unlimited quota
	if q, _ := dao.FindRecordById("storage_quotas", "quota_foss_0001"); q == nil {
		rec := models.NewRecord(quotasCol)
		rec.SetId("quota_foss_0001")
		rec.Set("name", "FOSS Unlimited")
		rec.Set("quota", 1099511627776) // 1 TB
		rec.Set("usage", 0)
		rec.Set("maxFileSize", 262144000) // 250 MB
		rec.Set("max_file_size", 262144000) // 250 MB
		rec.Set("metered", false)
		_ = dao.SaveRecord(rec)
	} else {
		if q.GetInt("max_file_size") == 0 {
			q.Set("max_file_size", 262144000)
			_ = dao.SaveRecord(q)
		}
	}

	// 4. relays collection
	relaysCol, err := dao.FindCollectionByNameOrId("relays")
	if err != nil {
		log.Println("[Migration] Creating 'relays' collection...")
		authRule := types.Pointer("@request.auth.id != ''")
		maxOne := 1
		relaysCol = &models.Collection{
			Name:       "relays",
			Type:       models.CollectionTypeBase,
			ListRule:   authRule,
			ViewRule:   authRule,
			CreateRule: authRule,
			UpdateRule: authRule,
			DeleteRule: authRule,
			Indexes: types.JsonArray[string]{
				"CREATE UNIQUE INDEX `idx_relays_guid` ON `relays` (`guid`)",
			},
			Schema: schema.NewSchema(
				&schema.SchemaField{
					Name:     "guid",
					Type:     schema.FieldTypeText,
					Required: true,
				},
				&schema.SchemaField{
					Name:     "name",
					Type:     schema.FieldTypeText,
					Required: false,
				},
				&schema.SchemaField{
					Name: "path",
					Type: schema.FieldTypeText,
				},
				&schema.SchemaField{
					Name: "version",
					Type: schema.FieldTypeNumber,
				},
				&schema.SchemaField{
					Name: "userLimit",
					Type: schema.FieldTypeNumber,
				},
				&schema.SchemaField{
					Name: "provider",
					Type: schema.FieldTypeRelation,
					Options: &schema.RelationOptions{
						CollectionId:  providersCol.Id,
						MaxSelect:     &maxOne,
						CascadeDelete: false,
					},
				},
				&schema.SchemaField{
					Name: "storage_quota",
					Type: schema.FieldTypeRelation,
					Options: &schema.RelationOptions{
						CollectionId:  quotasCol.Id,
						MaxSelect:     &maxOne,
						CascadeDelete: false,
					},
				},
				&schema.SchemaField{
					Name: "plan",
					Type: schema.FieldTypeText,
				},
				&schema.SchemaField{
					Name: "cta",
					Type: schema.FieldTypeText,
				},
			),
		}
		if err := dao.SaveCollection(relaysCol); err != nil {
			return err
		}
	} else {
		// Ensure name is not required for existing relays collection
		if nameField := relaysCol.Schema.GetFieldByName("name"); nameField != nil && nameField.Required {
			nameField.Required = false
			if err := dao.SaveCollection(relaysCol); err != nil {
				log.Printf("[Migration] Warning updating relays name field: %v\n", err)
			}
		}

		if relaysCol.Schema.GetFieldByName("version") == nil {
			relaysCol.Schema.AddField(&schema.SchemaField{
				Name: "version",
				Type: schema.FieldTypeNumber,
			})
			if err := dao.SaveCollection(relaysCol); err != nil {
				log.Printf("[Migration] Warning adding version field to relays: %v\n", err)
			}
		}

		// Ensure all existing relays have version >= 1
		existingRelays, _ := dao.FindRecordsByFilter("relays", "version = 0 || version = null", "", 0, 0)
		for _, r := range existingRelays {
			r.Set("version", 1)
			_ = dao.SaveRecord(r)
		}

		// Enforce tenant scoping: users only list relays they participate in
		relaysCol.ListRule = types.Pointer("@request.auth.id != '' && relay_roles_via_relay.user ?= @request.auth.id")
		if err := dao.SaveCollection(relaysCol); err != nil {
			log.Printf("[Migration] Warning updating relays ListRule: %v\n", err)
		}
	}

	// 5. shared_folders collection
	foldersCol, err := dao.FindCollectionByNameOrId("shared_folders")
	if err != nil {
		log.Println("[Migration] Creating 'shared_folders' collection...")
		authRule := types.Pointer("@request.auth.id != ''")
		maxOne := 1
		foldersCol = &models.Collection{
			Name:       "shared_folders",
			Type:       models.CollectionTypeBase,
			ListRule:   authRule,
			ViewRule:   authRule,
			CreateRule: authRule,
			UpdateRule: authRule,
			DeleteRule: authRule,
			Indexes: types.JsonArray[string]{
				"CREATE UNIQUE INDEX `idx_shared_folders_guid` ON `shared_folders` (`guid`)",
			},
			Schema: schema.NewSchema(
				&schema.SchemaField{
					Name:     "guid",
					Type:     schema.FieldTypeText,
					Required: true,
				},
				&schema.SchemaField{
					Name:     "name",
					Type:     schema.FieldTypeText,
					Required: true,
				},
				&schema.SchemaField{
					Name: "relay",
					Type: schema.FieldTypeRelation,
					Options: &schema.RelationOptions{
						CollectionId:  relaysCol.Id,
						MaxSelect:     &maxOne,
						CascadeDelete: true,
					},
				},
				&schema.SchemaField{
					Name: "creator",
					Type: schema.FieldTypeRelation,
					Options: &schema.RelationOptions{
						CollectionId:  usersCol.Id,
						MaxSelect:     &maxOne,
						CascadeDelete: false,
					},
				},
				&schema.SchemaField{
					Name: "private",
					Type: schema.FieldTypeBool,
				},
			),
		}
		if err := dao.SaveCollection(foldersCol); err != nil {
			return err
		}
	}

	// 6. relay_roles collection
	relayRolesCol, err := dao.FindCollectionByNameOrId("relay_roles")
	if err != nil {
		log.Println("[Migration] Creating 'relay_roles' collection...")
		authRule := types.Pointer("@request.auth.id != ''")
		maxOne := 1
		relayRolesCol = &models.Collection{
			Name:       "relay_roles",
			Type:       models.CollectionTypeBase,
			ListRule:   authRule,
			ViewRule:   authRule,
			CreateRule: authRule,
			UpdateRule: authRule,
			DeleteRule: authRule,
			Indexes: types.JsonArray[string]{
				"CREATE UNIQUE INDEX `idx_relay_roles_user_relay` ON `relay_roles` (`user`, `relay`)",
			},
			Schema: schema.NewSchema(
				&schema.SchemaField{
					Name: "user",
					Type: schema.FieldTypeRelation,
					Options: &schema.RelationOptions{
						CollectionId:  usersCol.Id,
						MaxSelect:     &maxOne,
						CascadeDelete: true,
					},
				},
				&schema.SchemaField{
					Name: "relay",
					Type: schema.FieldTypeRelation,
					Options: &schema.RelationOptions{
						CollectionId:  relaysCol.Id,
						MaxSelect:     &maxOne,
						CascadeDelete: true,
					},
				},
				&schema.SchemaField{
					Name: "role",
					Type: schema.FieldTypeRelation,
					Options: &schema.RelationOptions{
						CollectionId:  rolesCol.Id,
						MaxSelect:     &maxOne,
						CascadeDelete: false,
					},
				},
			),
		}
		if err := dao.SaveCollection(relayRolesCol); err != nil {
			return err
		}
	} else {
		relayRolesCol.ListRule = types.Pointer("@request.auth.id != '' && (user = @request.auth.id || relay.relay_roles_via_relay.user ?= @request.auth.id)")
		if err := dao.SaveCollection(relayRolesCol); err != nil {
			log.Printf("[Migration] Warning updating relay_roles ListRule: %v\n", err)
		}
	}

	// 7. shared_folder_roles collection
	sfRolesCol, err := dao.FindCollectionByNameOrId("shared_folder_roles")
	if err != nil {
		log.Println("[Migration] Creating 'shared_folder_roles' collection...")
		authRule := types.Pointer("@request.auth.id != ''")
		maxOne := 1
		sfRolesCol = &models.Collection{
			Name:       "shared_folder_roles",
			Type:       models.CollectionTypeBase,
			ListRule:   authRule,
			ViewRule:   authRule,
			CreateRule: authRule,
			UpdateRule: authRule,
			DeleteRule: authRule,
			Indexes: types.JsonArray[string]{
				"CREATE UNIQUE INDEX `idx_shared_folder_roles_user_sf` ON `shared_folder_roles` (`user`, `shared_folder`)",
			},
			Schema: schema.NewSchema(
				&schema.SchemaField{
					Name: "user",
					Type: schema.FieldTypeRelation,
					Options: &schema.RelationOptions{
						CollectionId:  usersCol.Id,
						MaxSelect:     &maxOne,
						CascadeDelete: true,
					},
				},
				&schema.SchemaField{
					Name: "shared_folder",
					Type: schema.FieldTypeRelation,
					Options: &schema.RelationOptions{
						CollectionId:  foldersCol.Id,
						MaxSelect:     &maxOne,
						CascadeDelete: true,
					},
				},
				&schema.SchemaField{
					Name: "role",
					Type: schema.FieldTypeRelation,
					Options: &schema.RelationOptions{
						CollectionId:  rolesCol.Id,
						MaxSelect:     &maxOne,
						CascadeDelete: false,
					},
				},
			),
		}
		if err := dao.SaveCollection(sfRolesCol); err != nil {
			return err
		}
	}

	// 8. relay_invitations collection
	invitesCol, err := dao.FindCollectionByNameOrId("relay_invitations")
	if err != nil {
		log.Println("[Migration] Creating 'relay_invitations' collection...")
		authRule := types.Pointer("@request.auth.id != ''")
		maxOne := 1
		invitesCol = &models.Collection{
			Name:       "relay_invitations",
			Type:       models.CollectionTypeBase,
			ListRule:   authRule,
			ViewRule:   authRule,
			CreateRule: authRule,
			UpdateRule: authRule,
			DeleteRule: authRule,
			Indexes: types.JsonArray[string]{
				"CREATE UNIQUE INDEX `idx_relay_invitations_key` ON `relay_invitations` (`key`)",
			},
			Schema: schema.NewSchema(
				&schema.SchemaField{
					Name: "relay",
					Type: schema.FieldTypeRelation,
					Options: &schema.RelationOptions{
						CollectionId:  relaysCol.Id,
						MaxSelect:     &maxOne,
						CascadeDelete: true,
					},
				},
				&schema.SchemaField{
					Name: "role",
					Type: schema.FieldTypeRelation,
					Options: &schema.RelationOptions{
						CollectionId:  rolesCol.Id,
						MaxSelect:     &maxOne,
						CascadeDelete: false,
					},
				},
				&schema.SchemaField{
					Name:     "key",
					Type:     schema.FieldTypeText,
					Required: true,
				},
				&schema.SchemaField{
					Name: "enabled",
					Type: schema.FieldTypeBool,
				},
			),
		}
		if err := dao.SaveCollection(invitesCol); err != nil {
			return err
		}
	}

	// 9. subscriptions collection
	subsCol, err := dao.FindCollectionByNameOrId("subscriptions")
	if err != nil {
		log.Println("[Migration] Creating 'subscriptions' collection...")
		authRule := types.Pointer("@request.auth.id != ''")
		maxOne := 1
		subsCol = &models.Collection{
			Name:       "subscriptions",
			Type:       models.CollectionTypeBase,
			ListRule:   authRule,
			ViewRule:   authRule,
			CreateRule: authRule,
			UpdateRule: authRule,
			DeleteRule: authRule,
			Schema: schema.NewSchema(
				&schema.SchemaField{
					Name: "user",
					Type: schema.FieldTypeRelation,
					Options: &schema.RelationOptions{
						CollectionId:  usersCol.Id,
						MaxSelect:     &maxOne,
						CascadeDelete: true,
					},
				},
				&schema.SchemaField{
					Name: "relay",
					Type: schema.FieldTypeRelation,
					Options: &schema.RelationOptions{
						CollectionId:  relaysCol.Id,
						MaxSelect:     &maxOne,
						CascadeDelete: true,
					},
				},
				&schema.SchemaField{
					Name: "active",
					Type: schema.FieldTypeBool,
				},
				&schema.SchemaField{
					Name: "stripe_cancel_at",
					Type: schema.FieldTypeNumber,
				},
				&schema.SchemaField{
					Name: "stripe_quantity",
					Type: schema.FieldTypeNumber,
				},
				&schema.SchemaField{
					Name: "token",
					Type: schema.FieldTypeText,
				},
			),
		}
		if err := dao.SaveCollection(subsCol); err != nil {
			return err
		}
	}

	// 10. devices collection
	devicesCol, err := dao.FindCollectionByNameOrId("devices")
	if err != nil {
		log.Println("[Migration] Creating 'devices' collection...")
		authRule := types.Pointer("@request.auth.id != ''")
		maxOne := 1
		devicesCol = &models.Collection{
			Name:       "devices",
			Type:       models.CollectionTypeBase,
			ListRule:   authRule,
			ViewRule:   authRule,
			CreateRule: authRule,
			UpdateRule: authRule,
			DeleteRule: authRule,
			Schema: schema.NewSchema(
				&schema.SchemaField{
					Name: "name",
					Type: schema.FieldTypeText,
				},
				&schema.SchemaField{
					Name: "platform",
					Type: schema.FieldTypeText,
				},
				&schema.SchemaField{
					Name: "user",
					Type: schema.FieldTypeRelation,
					Options: &schema.RelationOptions{
						CollectionId:  usersCol.Id,
						MaxSelect:     &maxOne,
						CascadeDelete: true,
					},
				},
			),
		}
		if err := dao.SaveCollection(devicesCol); err != nil {
			return err
		}
	}

	// 11. vaults collection
	vaultsCol, err := dao.FindCollectionByNameOrId("vaults")
	if err != nil {
		log.Println("[Migration] Creating 'vaults' collection...")
		authRule := types.Pointer("@request.auth.id != ''")
		maxOne := 1
		vaultsCol = &models.Collection{
			Name:       "vaults",
			Type:       models.CollectionTypeBase,
			ListRule:   authRule,
			ViewRule:   authRule,
			CreateRule: authRule,
			UpdateRule: authRule,
			DeleteRule: authRule,
			Schema: schema.NewSchema(
				&schema.SchemaField{
					Name: "device",
					Type: schema.FieldTypeText,
				},
				&schema.SchemaField{
					Name: "user",
					Type: schema.FieldTypeRelation,
					Options: &schema.RelationOptions{
						CollectionId:  usersCol.Id,
						MaxSelect:     &maxOne,
						CascadeDelete: true,
					},
				},
			),
		}
		if err := dao.SaveCollection(vaultsCol); err != nil {
			return err
		}
	}

	// Clean up automated test records (test_*@example.com)
	testUsers, _ := dao.FindRecordsByFilter("users", "email ~ 'test_'", "", 0, 0)
	for _, u := range testUsers {
		rRoles, _ := dao.FindRecordsByFilter("relay_roles", "user = {:u}", "", 0, 0, dbx.Params{"u": u.Id})
		for _, rr := range rRoles {
			relayID := rr.GetString("relay")
			_ = dao.DeleteRecord(rr)
			otherRoles, _ := dao.FindRecordsByFilter("relay_roles", "relay = {:r}", "", 0, 0, dbx.Params{"r": relayID})
			if len(otherRoles) == 0 {
				if rRec, err := dao.FindRecordById("relays", relayID); err == nil {
					_ = dao.DeleteRecord(rRec)
				}
			}
		}
		_ = dao.DeleteRecord(u)
	}

	return nil
}
