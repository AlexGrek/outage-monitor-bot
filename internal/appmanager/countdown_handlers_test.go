package appmanager

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestCountdownCRUD exercises create, list, update, and delete via the API.
func TestCountdownCRUD(t *testing.T) {
	am, _, cleanup := setupTestAppManager(t)
	defer cleanup()

	const key = "test-api-key"

	// Create
	body := `{
		"name": "New Year",
		"target_date": "2099-01-01",
		"notify_time": "09:00",
		"timezone": "UTC",
		"message_template": "{} days until New Year",
		"chat_ids": [123],
		"webhook_ids": []
	}`
	rec := makeRequest(t, am, http.MethodPost, "/countdowns", body, key)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}

	var created map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("create: bad response: %v", err)
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("create: missing id in response")
	}

	// List
	rec = makeRequest(t, am, http.MethodGet, "/countdowns", "", key)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", rec.Code)
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("list: bad response: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list: expected 1 countdown, got %d", len(list))
	}

	// Update
	updateBody := `{
		"name": "New Year Party",
		"target_date": "2099-01-01",
		"notify_time": "10:30",
		"timezone": "Europe/Kyiv",
		"message_template": "Only {} days left!",
		"enabled": false,
		"chat_ids": [123, 456],
		"webhook_ids": []
	}`
	rec = makeRequest(t, am, http.MethodPut, "/countdowns/"+id, updateBody, key)
	if rec.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var updated map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("update: bad response: %v", err)
	}
	if updated["name"] != "New Year Party" {
		t.Errorf("update: name not updated, got %v", updated["name"])
	}
	if updated["enabled"] != false {
		t.Errorf("update: enabled not updated, got %v", updated["enabled"])
	}

	// Delete
	rec = makeRequest(t, am, http.MethodDelete, "/countdowns/"+id, "", key)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d", rec.Code)
	}

	// Confirm gone
	rec = makeRequest(t, am, http.MethodGet, "/countdowns", "", key)
	var afterDelete []map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &afterDelete)
	if len(afterDelete) != 0 {
		t.Fatalf("after delete: expected 0 countdowns, got %d", len(afterDelete))
	}
}

// TestCountdownValidation checks that invalid inputs are rejected.
func TestCountdownValidation(t *testing.T) {
	am, _, cleanup := setupTestAppManager(t)
	defer cleanup()

	const key = "test-api-key"

	tests := []struct {
		name string
		body string
	}{
		{"missing name", `{"target_date":"2099-01-01","notify_time":"09:00","timezone":"UTC","message_template":"{}"}`},
		{"bad date", `{"name":"x","target_date":"01-01-2099","notify_time":"09:00","timezone":"UTC","message_template":"{}"}`},
		{"bad time", `{"name":"x","target_date":"2099-01-01","notify_time":"9am","timezone":"UTC","message_template":"{}"}`},
		{"bad timezone", `{"name":"x","target_date":"2099-01-01","notify_time":"09:00","timezone":"Mars/Olympus","message_template":"{}"}`},
		{"empty template", `{"name":"x","target_date":"2099-01-01","notify_time":"09:00","timezone":"UTC","message_template":""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := makeRequest(t, am, http.MethodPost, "/countdowns", tt.body, key)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d (%s)", rec.Code, rec.Body.String())
			}
		})
	}
}

// TestCountdownRequiresAuth verifies the endpoints are behind the API key.
func TestCountdownRequiresAuth(t *testing.T) {
	am, _, cleanup := setupTestAppManager(t)
	defer cleanup()

	rec := makeRequest(t, am, http.MethodGet, "/countdowns", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without API key, got %d", rec.Code)
	}
}
