package alert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// WebhookAlerter sends alerts as JSON POST to a webhook URL.
type WebhookAlerter struct {
	url      string
	client   *http.Client
	throttle *ThrottleMap
}

// NewWebhookAlerter creates a new webhook alerter.
func NewWebhookAlerter(url string, throttleMin int) *WebhookAlerter {
	return &WebhookAlerter{
		url: url,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		throttle: NewThrottleMap(throttleMin),
	}
}

func (w *WebhookAlerter) Name() string { return "webhook" }

func (w *WebhookAlerter) Send(alert Alert) error {
	if !w.throttle.ShouldSend(alert.RuleID + ":" + alert.SourceIP) {
		return nil
	}

	body, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("marshal alert: %w", err)
	}

	resp, err := w.client.Post(w.url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("alert webhook send failed: %v", err)
		return err
	}
	resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("alert webhook returned %d", resp.StatusCode)
	}
	return nil
}
