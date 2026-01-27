package notify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewNotifier(t *testing.T) {
	cfg := Config{
		Enabled:         true,
		NotifyOnSuccess: true,
		NotifyOnError:   true,
		Providers: []ProviderConfig{
			{Type: "slack", URL: "https://hooks.slack.com/test"},
			{Type: "discord", URL: "https://discord.com/api/webhooks/test"},
			{Type: "webhook", URL: "https://example.com/webhook"},
			{Type: "ntfy", URL: "https://ntfy.sh", Options: map[string]string{"topic": "test"}},
		},
	}

	notifier := NewNotifier(cfg)

	if !notifier.enabled {
		t.Error("enabled should be true")
	}

	if !notifier.onSuccess {
		t.Error("onSuccess should be true")
	}

	if !notifier.onError {
		t.Error("onError should be true")
	}

	if len(notifier.providers) != 4 {
		t.Errorf("len(providers) = %d, want 4", len(notifier.providers))
	}
}

func TestNotifierDisabled(t *testing.T) {
	notifier := &Notifier{
		enabled: false,
	}

	// Should not error when disabled
	err := notifier.NotifySuccess("Test", "Body", nil)
	if err != nil {
		t.Errorf("NotifySuccess() error = %v, want nil", err)
	}

	err = notifier.NotifyError("Test", "Body", nil, nil)
	if err != nil {
		t.Errorf("NotifyError() error = %v, want nil", err)
	}
}

func TestNotifySuccess(t *testing.T) {
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}

		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("Failed to decode payload: %v", err)
		}
		receivedPayload = payload

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := &Notifier{
		enabled:   true,
		onSuccess: true,
		providers: []Provider{&WebhookProvider{URL: server.URL}},
	}

	err := notifier.NotifySuccess("Backup Complete", "Successfully backed up 100 files", map[string]string{
		"files": "100",
		"size":  "1GB",
	})

	if err != nil {
		t.Fatalf("NotifySuccess() error = %v", err)
	}

	if receivedPayload == nil {
		t.Fatal("No payload received")
	}

	if receivedPayload["title"] != "Backup Complete" {
		t.Errorf("title = %v, want %q", receivedPayload["title"], "Backup Complete")
	}

	if receivedPayload["status"] != "success" {
		t.Errorf("status = %v, want %q", receivedPayload["status"], "success")
	}
}

func TestNotifyError(t *testing.T) {
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("Failed to decode payload: %v", err)
		}
		receivedPayload = payload
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := &Notifier{
		enabled:   true,
		onError:   true,
		providers: []Provider{&WebhookProvider{URL: server.URL}},
	}

	testErr := errorString("test error message")
	err := notifier.NotifyError("Backup Failed", "Backup operation failed", testErr, nil)

	if err != nil {
		t.Fatalf("NotifyError() error = %v", err)
	}

	if receivedPayload == nil {
		t.Fatal("No payload received")
	}

	if receivedPayload["status"] != "error" {
		t.Errorf("status = %v, want %q", receivedPayload["status"], "error")
	}

	details, ok := receivedPayload["details"].(map[string]interface{})
	if !ok {
		t.Fatal("details not found or wrong type")
	}

	if details["error"] != "test error message" {
		t.Errorf("details.error = %v, want %q", details["error"], "test error message")
	}
}

type errorString string

func (e errorString) Error() string {
	return string(e)
}

func TestSlackProvider(t *testing.T) {
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		json.NewDecoder(r.Body).Decode(&payload)
		receivedPayload = payload
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := &SlackProvider{URL: server.URL}

	msg := &Message{
		Title:     "Test",
		Body:      "Test body",
		Status:    "success",
		Timestamp: time.Now(),
	}

	err := provider.Send(msg)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if provider.Name() != "slack" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "slack")
	}

	attachments, ok := receivedPayload["attachments"].([]interface{})
	if !ok || len(attachments) == 0 {
		t.Fatal("attachments not found")
	}

	attachment := attachments[0].(map[string]interface{})
	if attachment["title"] != "Test" {
		t.Errorf("title = %v, want %q", attachment["title"], "Test")
	}
}

func TestDiscordProvider(t *testing.T) {
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		json.NewDecoder(r.Body).Decode(&payload)
		receivedPayload = payload
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := &DiscordProvider{URL: server.URL}

	msg := &Message{
		Title:     "Test",
		Body:      "Test body",
		Status:    "error",
		Timestamp: time.Now(),
	}

	err := provider.Send(msg)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if provider.Name() != "discord" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "discord")
	}

	embeds, ok := receivedPayload["embeds"].([]interface{})
	if !ok || len(embeds) == 0 {
		t.Fatal("embeds not found")
	}
}

func TestNtfyProvider(t *testing.T) {
	var receivedTitle, receivedPriority string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedTitle = r.Header.Get("Title")
		receivedPriority = r.Header.Get("Priority")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := &NtfyProvider{URL: server.URL, Topic: "test"}

	msg := &Message{
		Title:     "Test Title",
		Body:      "Test body",
		Status:    "error",
		Timestamp: time.Now(),
	}

	err := provider.Send(msg)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if provider.Name() != "ntfy" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "ntfy")
	}

	if receivedTitle != "Test Title" {
		t.Errorf("Title header = %q, want %q", receivedTitle, "Test Title")
	}

	if receivedPriority != "high" {
		t.Errorf("Priority header = %q, want %q", receivedPriority, "high")
	}
}

func TestWebhookProviderWithHeaders(t *testing.T) {
	var receivedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := &WebhookProvider{
		URL: server.URL,
		Headers: map[string]string{
			"Authorization": "Bearer test-token",
		},
	}

	msg := &Message{
		Title:     "Test",
		Body:      "Test body",
		Status:    "success",
		Timestamp: time.Now(),
	}

	err := provider.Send(msg)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if receivedAuth != "Bearer test-token" {
		t.Errorf("Authorization header = %q, want %q", receivedAuth, "Bearer test-token")
	}
}

func TestWebhookProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	provider := &WebhookProvider{URL: server.URL}

	msg := &Message{
		Title:     "Test",
		Body:      "Test body",
		Status:    "success",
		Timestamp: time.Now(),
	}

	err := provider.Send(msg)
	if err == nil {
		t.Error("Expected error for 500 response")
	}
}

func TestCreateProviderUnknown(t *testing.T) {
	cfg := ProviderConfig{
		Type: "unknown",
		URL:  "https://example.com",
	}

	provider := createProvider(cfg)
	if provider != nil {
		t.Error("Expected nil for unknown provider type")
	}
}

func TestBuildSlackFields(t *testing.T) {
	details := map[string]string{
		"files": "100",
		"size":  "1GB",
	}

	fields := buildSlackFields(details)
	if len(fields) != 2 {
		t.Errorf("len(fields) = %d, want 2", len(fields))
	}

	// Check structure
	for _, field := range fields {
		if _, ok := field["title"]; !ok {
			t.Error("field missing 'title'")
		}
		if _, ok := field["value"]; !ok {
			t.Error("field missing 'value'")
		}
		if _, ok := field["short"]; !ok {
			t.Error("field missing 'short'")
		}
	}
}
