package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"slack-proxy/config"
)

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(content)
	f.Close()
	return f.Name()
}

func TestLoad_Valid(t *testing.T) {
	path := writeTemp(t, `
server:
  port: 9090
routes:
  - slack_path: /hook/a
    dingtalk:
      webhook: https://oapi.dingtalk.com/robot/send?access_token=TOKEN
      secret: SECRET
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("port = %d, want 9090", cfg.Server.Port)
	}
	if len(cfg.Routes) != 1 {
		t.Fatalf("routes count = %d, want 1", len(cfg.Routes))
	}
	r := cfg.Routes[0]
	if r.SlackPath != "/hook/a" {
		t.Errorf("slack_path = %q, want /hook/a", r.SlackPath)
	}
	if r.DingTalk.Secret != "SECRET" {
		t.Errorf("secret = %q, want SECRET", r.DingTalk.Secret)
	}
}

func TestLoad_DefaultPort(t *testing.T) {
	path := writeTemp(t, `
routes:
  - slack_path: /hook/a
    dingtalk:
      webhook: https://example.com
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("port = %d, want 8080 (default)", cfg.Server.Port)
	}
}

func TestLoad_MultipleRoutes(t *testing.T) {
	path := writeTemp(t, `
routes:
  - slack_path: /hook/a
    dingtalk:
      webhook: https://example.com/a
  - slack_path: /hook/b
    dingtalk:
      webhook: https://example.com/b
      secret: s2
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Routes) != 2 {
		t.Errorf("routes count = %d, want 2", len(cfg.Routes))
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := config.Load(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeTemp(t, `:::invalid yaml:::`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestLoad_NoRoutes(t *testing.T) {
	path := writeTemp(t, `server:\n  port: 8080\n`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for empty routes, got nil")
	}
}

func TestLoad_MissingSlackPath(t *testing.T) {
	path := writeTemp(t, `
routes:
  - dingtalk:
      webhook: https://example.com
`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for missing slack_path, got nil")
	}
}

func TestLoad_MissingWebhook(t *testing.T) {
	path := writeTemp(t, `
routes:
  - slack_path: /hook/a
    dingtalk:
      secret: only-secret
`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for missing webhook, got nil")
	}
}
