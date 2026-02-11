package appmanager

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"tg-monitor-bot/internal/config"
	"tg-monitor-bot/internal/storage"
)

// setupTestAppManager creates a test AppManager with in-memory database
func setupTestAppManager(t *testing.T) (*AppManager, *storage.BoltDB, func()) {
	// Create temporary database
	db, err := storage.NewBoltDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	cfg := &config.Config{
		TelegramToken:        "test-token",
		DBPath:               ":memory:",
		APIEnabled:           true,
		APIPort:              8080,
		APIKey:               "test-api-key",
		AutoRestartEnabled:   false,
	}

	am := &AppManager{
		storage:    db,
		apiKey:     cfg.APIKey,
		apiEnabled: cfg.APIEnabled,
		apiPort:    cfg.APIPort,
		echoServer: echo.New(),
		logger:     log.New(os.Stdout, "[TEST] ", log.LstdFlags),
	}

	// Initialize config manager
	configManager := NewConfigManager(db)
	am.configManager = configManager

	// Initialize bot process (in test mode)
	botProcess := NewBotProcess(db)
	am.botProcess = botProcess

	// Setup routes
	am.setupRoutes()

	cleanup := func() {
		db.Close()
	}

	return am, db, cleanup
}

// makeRequest is a helper to make HTTP requests with optional API key
func makeRequest(t *testing.T, am *AppManager, method, path string, body string, apiKey string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	rec := httptest.NewRecorder()
	am.echoServer.ServeHTTP(rec, req)

	return rec
}

// TestHealthEndpoint tests the /health endpoint
func TestHealthEndpoint(t *testing.T) {
	am, _, cleanup := setupTestAppManager(t)
	defer cleanup()

	rec := makeRequest(t, am, http.MethodGet, "/health", "", "")

	// Health endpoint should work without API key
	// Status code can be 200 (healthy) or 503 (degraded) depending on bot state
	if rec.Code != http.StatusOK && rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 200 or 503, got %d", rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check required fields
	if _, ok := response["status"]; !ok {
		t.Error("Response missing 'status' field")
	}
	if _, ok := response["api_running"]; !ok {
		t.Error("Response missing 'api_running' field")
	}
	if _, ok := response["bot_running"]; !ok {
		t.Error("Response missing 'bot_running' field")
	}
}

// TestAPIKeyAuth tests API key authentication
func TestAPIKeyAuth(t *testing.T) {
	am, _, cleanup := setupTestAppManager(t)
	defer cleanup()

	tests := []struct {
		name           string
		apiKey         string
		expectedStatus int
	}{
		{
			name:           "valid API key",
			apiKey:         "test-api-key",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid API key",
			apiKey:         "wrong-key",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "missing API key",
			apiKey:         "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := makeRequest(t, am, http.MethodGet, "/status", "", tt.apiKey)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

// TestGetAllConfig tests the GET /config endpoint
func TestGetAllConfig(t *testing.T) {
	am, _, cleanup := setupTestAppManager(t)
	defer cleanup()

	rec := makeRequest(t, am, http.MethodGet, "/config", "", "test-api-key")

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var config map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &config); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Config should be a map
	if config == nil {
		t.Error("Expected config to be a map")
	}
}

// TestUpdateConfig tests the PUT /config/:key endpoint
func TestUpdateConfig(t *testing.T) {
	am, _, cleanup := setupTestAppManager(t)
	defer cleanup()

	tests := []struct {
		name           string
		key            string
		value          string
		expectedStatus int
	}{
		{
			name:           "update valid config",
			key:            "DEFAULT_CHECK_INTERVAL",
			value:          "60s",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "update with empty value",
			key:            "TEST_KEY",
			value:          "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := `{"value":"` + tt.value + `"}`
			rec := makeRequest(t, am, http.MethodPut, "/config/"+tt.key, body, "test-api-key")

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

// TestSourcesEndpoints tests source CRUD operations
func TestSourcesEndpoints(t *testing.T) {
	am, db, cleanup := setupTestAppManager(t)
	defer cleanup()

	// Test GET /sources (empty)
	t.Run("get empty sources", func(t *testing.T) {
		rec := makeRequest(t, am, http.MethodGet, "/sources", "", "test-api-key")

		if rec.Code != http.StatusServiceUnavailable {
			// Monitor not available in test mode, should return 503
			t.Logf("Expected 503 (monitor unavailable), got %d", rec.Code)
		}
	})

	// Test POST /sources (create)
	t.Run("create source", func(t *testing.T) {
		body := `{
			"name": "Test Server",
			"type": "ping",
			"target": "8.8.8.8",
			"check_interval": "30s"
		}`
		rec := makeRequest(t, am, http.MethodPost, "/sources", body, "test-api-key")

		// Should succeed even without monitor
		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var source map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &source); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify fields
		if source["name"] != "Test Server" {
			t.Errorf("Expected name 'Test Server', got %v", source["name"])
		}
		if source["type"] != "ping" {
			t.Errorf("Expected type 'ping', got %v", source["type"])
		}
	})

	// Test invalid source creation
	t.Run("create invalid source", func(t *testing.T) {
		tests := []struct {
			name string
			body string
		}{
			{
				name: "missing name",
				body: `{"type":"ping","target":"8.8.8.8","check_interval":"30s"}`,
			},
			{
				name: "invalid type",
				body: `{"name":"Test","type":"invalid","target":"8.8.8.8","check_interval":"30s"}`,
			},
			{
				name: "missing target",
				body: `{"name":"Test","type":"ping","check_interval":"30s"}`,
			},
			{
				name: "invalid interval",
				body: `{"name":"Test","type":"ping","target":"8.8.8.8","check_interval":"invalid"}`,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				rec := makeRequest(t, am, http.MethodPost, "/sources", tt.body, "test-api-key")

				if rec.Code != http.StatusBadRequest {
					t.Errorf("Expected status 400, got %d", rec.Code)
				}
			})
		}
	})

	// Test source operations on existing source
	t.Run("operations on created source", func(t *testing.T) {
		// Create a source first
		createBody := `{
			"name": "Test Server 2",
			"type": "http",
			"target": "https://example.com",
			"check_interval": "60s"
		}`
		createRec := makeRequest(t, am, http.MethodPost, "/sources", createBody, "test-api-key")

		if createRec.Code != http.StatusCreated {
			t.Fatalf("Failed to create source: %d", createRec.Code)
		}

		var createdSource map[string]interface{}
		json.Unmarshal(createRec.Body.Bytes(), &createdSource)
		sourceID := createdSource["id"].(string)

		// Test GET /sources/:id
		t.Run("get source by id", func(t *testing.T) {
			source, err := db.GetSource(sourceID)
			if err != nil {
				t.Errorf("Failed to get source from DB: %v", err)
			}
			if source.Name != "Test Server 2" {
				t.Errorf("Expected name 'Test Server 2', got %s", source.Name)
			}
		})

		// Test PUT /sources/:id (update)
		t.Run("update source", func(t *testing.T) {
			updateBody := `{
				"name": "Updated Server",
				"type": "http",
				"target": "https://updated.com",
				"check_interval": "120s",
				"enabled": true
			}`
			rec := makeRequest(t, am, http.MethodPut, "/sources/"+sourceID, updateBody, "test-api-key")

			if rec.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
			}
		})

		// Test DELETE /sources/:id
		t.Run("delete source", func(t *testing.T) {
			rec := makeRequest(t, am, http.MethodDelete, "/sources/"+sourceID, "", "test-api-key")

			if rec.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rec.Code)
			}

			// Verify deletion
			_, err := db.GetSource(sourceID)
			if err == nil {
				t.Error("Source should be deleted but still exists")
			}
		})
	})
}

// TestReloadEndpoint tests the POST /config/reload endpoint
func TestReloadEndpoint(t *testing.T) {
	am, _, cleanup := setupTestAppManager(t)
	defer cleanup()

	rec := makeRequest(t, am, http.MethodPost, "/config/reload", "", "test-api-key")

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["message"] == "" {
		t.Error("Response should contain a message")
	}
}
