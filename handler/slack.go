package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"slack-proxy/config"
	"slack-proxy/dingtalk"
)

// Sender is a function that forwards a Slack message to DingTalk.
type Sender func(webhook, secret string, msg *dingtalk.SlackMessage) error

// NewSlackHandler returns an HTTP handler for a specific route.
// If sender is nil, dingtalk.Send is used.
func NewSlackHandler(route config.Route, sender Sender) http.HandlerFunc {
	if sender == nil {
		sender = dingtalk.Send
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var msg dingtalk.SlackMessage
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			log.Printf("[%s] decode body error: %v", route.SlackPath, err)
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}

		log.Printf("[%s] received message: %q", route.SlackPath, truncate(msg.Text, 80))

		if err := sender(route.DingTalk.Webhook, route.DingTalk.Secret, &msg); err != nil {
			log.Printf("[%s] forward to dingtalk failed: %v", route.SlackPath, err)
			http.Error(w, "failed to forward message", http.StatusBadGateway)
			return
		}

		log.Printf("[%s] forwarded successfully", route.SlackPath)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`)) //nolint:errcheck
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
