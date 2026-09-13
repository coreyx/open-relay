package handlers

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/open-relay/control-plane/crypto"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

type DocTokenRequest struct {
	DocID  string `json:"docId"`
	Relay  string `json:"relay"`
	Folder string `json:"folder"`
	Device string `json:"device"`
}

type FileTokenRequest struct {
	DocID         string `json:"docId"`
	Relay         string `json:"relay"`
	Folder        string `json:"folder"`
	Hash          string `json:"hash"`
	ContentType   string `json:"contentType"`
	ContentLength int64  `json:"contentLength"`
	Device        string `json:"device"`
}

type TokenResponse struct {
	URL           string `json:"url"`
	BaseURL       string `json:"baseUrl"`
	DocID         string `json:"docId"`
	Token         string `json:"token"`
	Authorization string `json:"authorization"` // "full" or "read_only"
	FileHash      string `json:"fileHash,omitempty"`
}

// RegisterTokenHandlers registers /token and /file-token endpoints.
func RegisterTokenHandlers(router *echo.Echo, app core.App, km *crypto.KeyManager) {
	// Document token endpoint
	router.POST("/token", func(c echo.Context) error {
			authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
			if authRecord == nil {
				return apis.NewUnauthorizedError("Authentication required", nil)
			}

			var req DocTokenRequest
			if err := c.Bind(&req); err != nil {
				return apis.NewBadRequestError("Invalid request body", err)
			}
			if req.DocID == "" || req.Relay == "" {
				return apis.NewBadRequestError("Missing docId or relay", nil)
			}

			// Verify user permission on relay
			isReadOnly, relayRec, providerRec, err := checkRelayAccess(app, authRecord.Id, req.Relay, req.Folder)
			if err != nil {
				return apis.NewForbiddenError(err.Error(), nil)
			}

			providerURL := getProviderURL(providerRec, c)
			channel := relayRec.GetString("guid")
			token, err := km.SignDocToken(req.DocID, authRecord.Id, providerURL, channel, isReadOnly, 3600)
			if err != nil {
				return apis.NewBadRequestError(fmt.Sprintf("Failed to generate token: %v", err), nil)
			}

			authStr := "full"
			if isReadOnly {
				authStr = "read_only"
			}

			wsURL := toWebSocketURL(providerURL) + "/d/" + req.DocID + "/ws"
			baseURL := strings.TrimRight(providerURL, "/") + "/d/" + req.DocID

			return c.JSON(http.StatusOK, TokenResponse{
				URL:           wsURL,
				BaseURL:       baseURL,
				DocID:         req.DocID,
				Token:         token,
				Authorization: authStr,
			})
		}, apis.ActivityLogger(app))

	// File token endpoint
	router.POST("/file-token", func(c echo.Context) error {
		authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if authRecord == nil {
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		var req FileTokenRequest
		if err := c.Bind(&req); err != nil {
			return apis.NewBadRequestError("Invalid request body", err)
		}
		if req.DocID == "" || req.Relay == "" || req.Hash == "" {
			return apis.NewBadRequestError("Missing docId, relay, or hash", nil)
		}

		isReadOnly, relayRec, providerRec, err := checkRelayAccess(app, authRecord.Id, req.Relay, req.Folder)
		if err != nil {
			return apis.NewForbiddenError(err.Error(), nil)
		}

		providerURL := getProviderURL(providerRec, c)
		channel := relayRec.GetString("guid")
		token, err := km.SignFileToken(req.DocID, req.Hash, authRecord.Id, providerURL, channel, isReadOnly, 3600)
		if err != nil {
			return apis.NewBadRequestError(fmt.Sprintf("Failed to generate file token: %v", err), nil)
		}

		authStr := "full"
		if isReadOnly {
			authStr = "read_only"
		}

		wsURL := toWebSocketURL(providerURL) + "/d/" + req.DocID + "/ws"
		baseURL := strings.TrimRight(providerURL, "/") + "/d/" + req.DocID

		return c.JSON(http.StatusOK, TokenResponse{
			URL:           wsURL,
			BaseURL:       baseURL,
			DocID:         req.DocID,
			Token:         token,
			Authorization: authStr,
			FileHash:      req.Hash,
		})
	}, apis.ActivityLogger(app))

	// Flags endpoint for client feature flags
	router.GET("/flags", func(c echo.Context) error {
		return c.JSON(http.StatusOK, []any{})
	}, apis.ActivityLogger(app))

	// WhoAmI endpoint
	router.GET("/whoami", func(c echo.Context) error {
		authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if authRecord == nil {
			return apis.NewUnauthorizedError("Authentication required", nil)
		}
		return c.JSON(http.StatusOK, map[string]any{
			"id":    authRecord.Id,
			"email": authRecord.Email(),
			"name":  authRecord.GetString("name"),
		})
	}, apis.ActivityLogger(app))

	// Check-host endpoint
	router.GET("/relay/:guid/check-host", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]any{
			"level":  "ok",
			"status": "Healthy",
		})
	}, apis.ActivityLogger(app))
}

// checkRelayAccess checks whether user has access to relay and folder, returning isReadOnly, relay, provider, and error.
func checkRelayAccess(app core.App, userID, relayID, folderID string) (bool, *models.Record, *models.Record, error) {
	dao := app.Dao()

	// Find the relay record (by PocketBase ID or by GUID)
	relayRec, err := dao.FindRecordById("relays", relayID)
	if err != nil {
		relayRec, _ = dao.FindFirstRecordByFilter("relays", "guid = {:guid}", dbx.Params{"guid": relayID})
	}
	if relayRec == nil {
		return false, nil, nil, errors.New("relay not found")
	}

	// Check relay_roles for this user using the relay's record ID
	roleID := ""
	relayRole, err := dao.FindFirstRecordByFilter("relay_roles", "user = {:user} && relay = {:relay}", dbx.Params{
		"user":  userID,
		"relay": relayRec.Id,
	})
	if err == nil && relayRole != nil {
		roleID = relayRole.GetString("role")
	}

	// If folder is specified, check folder-level access
	if folderID != "" {
		folderRec, _ := dao.FindRecordById("shared_folders", folderID)
		if folderRec == nil {
			folderRec, _ = dao.FindFirstRecordByFilter("shared_folders", "guid = {:guid}", dbx.Params{"guid": folderID})
		}
		if folderRec != nil {
			sfRole, err := dao.FindFirstRecordByFilter("shared_folder_roles", "user = {:user} && shared_folder = {:folder}", dbx.Params{
				"user":   userID,
				"folder": folderRec.Id,
			})
			if err == nil && sfRole != nil {
				roleID = sfRole.GetString("role")
			} else if folderRec.GetBool("private") {
				// For private folders, if user is not relay owner or folder creator, and has no folder role: deny access
				isOwner := roleID == "role_owner_001" || roleID == "2arnubkcv7jpce8" || folderRec.GetString("creator") == userID
				if !isOwner && roleID != "" {
					rRec, _ := dao.FindRecordById("roles", roleID)
					if rRec != nil && strings.ToLower(rRec.GetString("name")) == "owner" {
						isOwner = true
					}
				}
				if !isOwner {
					return false, nil, nil, errors.New("access denied: folder is private")
				}
			}
		}
	}

	if roleID == "" {
		return false, nil, nil, errors.New("access denied: no membership in this relay")
	}

	// Find the role record to get its name
	roleRec, err := dao.FindRecordById("roles", roleID)
	if err != nil {
		return false, nil, nil, errors.New("role definition not found")
	}

	roleName := strings.ToLower(roleRec.GetString("name"))
	isReadOnly := roleName == "reader"

	// Fetch provider
	providerID := relayRec.GetString("provider")
	var providerRec *models.Record
	if providerID != "" {
		providerRec, _ = dao.FindRecordById("providers", providerID)
	}

	return isReadOnly, relayRec, providerRec, nil
}

func getProviderURL(providerRec *models.Record, c echo.Context) string {
	rawURL := ""
	if providerRec != nil {
		rawURL = providerRec.GetString("url")
	}
	if rawURL == "" {
		scheme := "http"
		if c.IsTLS() || c.Request().Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		return fmt.Sprintf("%s://%s", scheme, c.Request().Host)
	}

	// If provider URL host is localhost or 127.0.0.1, but the request came from
	// a remote client (e.g. over LAN or Tailscale), dynamically adapt the host to the
	// request's hostname so the remote client can reach the data plane.
	parsed, err := url.Parse(rawURL)
	if err == nil && (parsed.Hostname() == "localhost" || parsed.Hostname() == "127.0.0.1") {
		reqHost := c.Request().Host
		if h, _, err := net.SplitHostPort(reqHost); err == nil {
			reqHost = h
		}
		if reqHost != "localhost" && reqHost != "127.0.0.1" && reqHost != "" {
			port := parsed.Port()
			if port != "" {
				parsed.Host = net.JoinHostPort(reqHost, port)
			} else {
				parsed.Host = reqHost
			}
			return parsed.String()
		}
	}

	return rawURL
}

func toWebSocketURL(httpURL string) string {
	if strings.HasPrefix(httpURL, "https://") {
		return "wss://" + strings.TrimPrefix(httpURL, "https://")
	}
	return "ws://" + strings.TrimPrefix(httpURL, "http://")
}
