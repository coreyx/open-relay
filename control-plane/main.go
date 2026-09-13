package main

import (
	"log"
	"os"

	"github.com/open-relay/control-plane/crypto"
	"github.com/open-relay/control-plane/handlers"
	"github.com/open-relay/control-plane/migrations"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	app := pocketbase.New()

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		dataDir := app.DataDir()
		if dataDir == "" {
			dataDir = "pb_data"
		}

		km, err := crypto.InitKeyManager(dataDir)
		if err != nil {
			log.Fatalf("Failed to initialize Ed25519 key manager: %v", err)
		}

		log.Printf("[Open-Relay] Initialized Key ID: %s", km.KeyID)
		log.Printf("[Open-Relay] Public Key (Base64): %s", km.PublicKeyBase64)

		if err := migrations.EnsureSchema(app); err != nil {
			log.Fatalf("Failed to run schema migrations: %v", err)
		}

		handlers.RegisterTokenHandlers(e.Router, app, km)
		handlers.RegisterInviteHandlers(e.Router, app)
		handlers.RegisterSelfHostHandler(e.Router, app, km)
		handlers.RegisterTemplateHandler(e.Router, app, km)
		handlers.RegisterRelayHooks(app, km)

		return nil
	})

	if err := app.Start(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
