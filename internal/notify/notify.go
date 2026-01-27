// Package notify provides notification capabilities for resticm
package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Provider represents a notification provider
type Provider interface {
	Send(message *Message) error
	Name() string
}

// Message represents a notification message
type Message struct {
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	Status    string            `json:"status"` // success, error, warning
	Timestamp time.Time         `json:"timestamp"`
	Details   map[string]string `json:"details,omitempty"`
}

// Config represents notification configuration
type Config struct {
	Enabled         bool             `yaml:"enabled"`
	NotifyOnSuccess bool             `yaml:"notify_on_success"`
	NotifyOnError   bool             `yaml:"notify_on_error"`
	Providers       []ProviderConfig `yaml:"providers"`
}

// ProviderConfig represents a provider configuration
type ProviderConfig struct {
	Type    string            `yaml:"type"`
	URL     string            `yaml:"url"`
	Options map[string]string `yaml:"options"`
}

// Notifier manages notifications
type Notifier struct {
	providers []Provider
	enabled   bool
	onSuccess bool
	onError   bool
}

// NewNotifier creates a new notifier from configuration
func NewNotifier(cfg Config) *Notifier {
	notifier := &Notifier{
		enabled:   cfg.Enabled,
		onSuccess: cfg.NotifyOnSuccess,
		onError:   cfg.NotifyOnError,
	}

	for _, pc := range cfg.Providers {
		provider := createProvider(pc)
		if provider != nil {
			notifier.providers = append(notifier.providers, provider)
		}
	}

	return notifier
}

func createProvider(cfg ProviderConfig) Provider {
	switch strings.ToLower(cfg.Type) {
	case "slack":
		return &SlackProvider{URL: cfg.URL}
	case "discord":
		return &DiscordProvider{URL: cfg.URL}
	case "webhook":
		return &WebhookProvider{
			URL:     cfg.URL,
			Headers: cfg.Options,
		}
	case "ntfy":
		topic := cfg.Options["topic"]
		return &NtfyProvider{URL: cfg.URL, Topic: topic}
	case "google", "googlechat", "google_chat":
		return &GoogleChatProvider{URL: cfg.URL}
	case "uptimekuma", "uptime_kuma", "uptime-kuma":
		return &UptimeKumaProvider{URL: cfg.URL}
	default:
		return nil
	}
}

// NotifySuccess sends a success notification
func (n *Notifier) NotifySuccess(title, body string, details map[string]string) error {
	if !n.enabled || !n.onSuccess {
		return nil
	}

	msg := &Message{
		Title:     title,
		Body:      body,
		Status:    "success",
		Timestamp: time.Now(),
		Details:   details,
	}

	return n.send(msg)
}

// NotifyError sends an error notification
func (n *Notifier) NotifyError(title, body string, err error, details map[string]string) error {
	if !n.enabled || !n.onError {
		return nil
	}

	if details == nil {
		details = make(map[string]string)
	}
	if err != nil {
		details["error"] = err.Error()
	}

	msg := &Message{
		Title:     title,
		Body:      body,
		Status:    "error",
		Timestamp: time.Now(),
		Details:   details,
	}

	return n.send(msg)
}

func (n *Notifier) send(msg *Message) error {
	var lastErr error
	for _, provider := range n.providers {
		if err := provider.Send(msg); err != nil {
			lastErr = fmt.Errorf("%s: %w", provider.Name(), err)
		}
	}
	return lastErr
}

// SlackProvider sends notifications to Slack
type SlackProvider struct {
	URL string
}

func (s *SlackProvider) Name() string {
	return "slack"
}

func (s *SlackProvider) Send(msg *Message) error {
	color := "#36a64f" // green
	if msg.Status == "error" {
		color = "#dc3545" // red
	}

	payload := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"color":  color,
				"title":  msg.Title,
				"text":   msg.Body,
				"footer": "resticm",
				"ts":     msg.Timestamp.Unix(),
				"fields": buildSlackFields(msg.Details),
			},
		},
	}

	return postJSON(s.URL, payload)
}

func buildSlackFields(details map[string]string) []map[string]interface{} {
	var fields []map[string]interface{}
	for k, v := range details {
		fields = append(fields, map[string]interface{}{
			"title": k,
			"value": v,
			"short": true,
		})
	}
	return fields
}

// DiscordProvider sends notifications to Discord
type DiscordProvider struct {
	URL string
}

func (d *DiscordProvider) Name() string {
	return "discord"
}

func (d *DiscordProvider) Send(msg *Message) error {
	color := 0x36a64f // green
	if msg.Status == "error" {
		color = 0xdc3545 // red
	}

	payload := map[string]interface{}{
		"embeds": []map[string]interface{}{
			{
				"title":       msg.Title,
				"description": msg.Body,
				"color":       color,
				"footer": map[string]string{
					"text": "resticm",
				},
				"timestamp": msg.Timestamp.Format(time.RFC3339),
			},
		},
	}

	return postJSON(d.URL, payload)
}

// WebhookProvider sends notifications to a generic webhook
type WebhookProvider struct {
	URL     string
	Headers map[string]string
}

func (w *WebhookProvider) Name() string {
	return "webhook"
}

func (w *WebhookProvider) Send(msg *Message) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", w.URL, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range w.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// NtfyProvider sends notifications to ntfy.sh
type NtfyProvider struct {
	URL   string
	Topic string
}

func (n *NtfyProvider) Name() string {
	return "ntfy"
}

func (n *NtfyProvider) Send(msg *Message) error {
	url := strings.TrimSuffix(n.URL, "/") + "/" + n.Topic

	priority := "default"
	tags := "white_check_mark"
	if msg.Status == "error" {
		priority = "high"
		tags = "x"
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(msg.Body))
	if err != nil {
		return err
	}

	req.Header.Set("Title", msg.Title)
	req.Header.Set("Priority", priority)
	req.Header.Set("Tags", tags)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ntfy returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func postJSON(url string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GoogleChatProvider sends notifications to Google Chat
type GoogleChatProvider struct {
	URL string
}

func (g *GoogleChatProvider) Name() string {
	return "google_chat"
}

func (g *GoogleChatProvider) Send(msg *Message) error {
	// Google Chat card format
	color := "#36a64f"
	icon := "✅"
	if msg.Status == "error" {
		color = "#dc3545"
		icon = "🚨"
	}

	// Build widgets for details
	var widgets []map[string]interface{}
	widgets = append(widgets, map[string]interface{}{
		"textParagraph": map[string]string{
			"text": msg.Body,
		},
	})

	for k, v := range msg.Details {
		widgets = append(widgets, map[string]interface{}{
			"keyValue": map[string]interface{}{
				"topLabel": k,
				"content":  v,
			},
		})
	}

	payload := map[string]interface{}{
		"cards": []map[string]interface{}{
			{
				"header": map[string]interface{}{
					"title":    icon + " " + msg.Title,
					"subtitle": "resticm backup",
				},
				"sections": []map[string]interface{}{
					{
						"widgets": widgets,
					},
				},
			},
		},
	}

	// Suppress unused variable warning
	_ = color

	return postJSON(g.URL, payload)
}

// UptimeKumaProvider sends heartbeat to Uptime Kuma
type UptimeKumaProvider struct {
	URL string
}

func (u *UptimeKumaProvider) Name() string {
	return "uptime_kuma"
}

func (u *UptimeKumaProvider) Send(msg *Message) error {
	// Uptime Kuma push monitor format
	// URL format: https://uptime.example.com/api/push/TOKEN?status=up&msg=OK
	status := "up"
	statusMsg := "OK"
	if msg.Status == "error" {
		status = "down"
		statusMsg = msg.Body
	}

	url := fmt.Sprintf("%s?status=%s&msg=%s", u.URL, status, statusMsg)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("uptime kuma returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
