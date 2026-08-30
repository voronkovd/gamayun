package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseClock(t *testing.T) {
	h, m, err := ParseClock("08:00")
	if err != nil || h != 8 || m != 0 {
		t.Fatalf("got %d:%d %v", h, m, err)
	}
	if _, _, err := ParseClock("25:00"); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadYAML(t *testing.T) {
	t.Setenv("TG_BOT_TOKEN", "")
	t.Setenv("TG_CHAT_ID", "")
	t.Setenv("VPS_TG_BOT_TOKEN", "")
	t.Setenv("VPS_TG_CHAT_ID", "")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
server_name: lab
telegram:
  bot_token: "tok"
  chat_id: 123456789
checks:
  interval: 30s
  nginx: off
  disk_pct_max: 70
  load15_max: 4.5
  containers:
    - a
    - b
alerts:
  fail_streak: 3
  escalation:
    - 1m
    - 2m
digest:
  at: "09:15"
paths:
  state: /tmp/state.json
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServerName != "lab" || cfg.DiskPctMax != 70 || cfg.Load15Max != 4.5 {
		t.Fatalf("basic fields: %+v", cfg)
	}
	if cfg.TGBotToken != "tok" || cfg.TGChatID != "123456789" {
		t.Fatalf("telegram: %q %q", cfg.TGBotToken, cfg.TGChatID)
	}
	if cfg.CheckInterval != 30*time.Second || cfg.FailStreak != 3 {
		t.Fatalf("timing: %+v", cfg)
	}
	if len(cfg.Escalation) != 2 || cfg.Escalation[0] != time.Minute {
		t.Fatalf("escalation: %v", cfg.Escalation)
	}
	if cfg.DigestHour != 9 || cfg.DigestMin != 15 {
		t.Fatalf("digest %d:%d", cfg.DigestHour, cfg.DigestMin)
	}
	if !cfg.NginxDisabled() {
		t.Fatal("nginx should be disabled")
	}
	if len(cfg.Containers) != 2 || cfg.Containers[0] != "a" {
		t.Fatalf("containers: %v", cfg.Containers)
	}
	if cfg.StatePath != "/tmp/state.json" {
		t.Fatalf("state %s", cfg.StatePath)
	}
}

func TestEnvOverridesTelegram(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("telegram:\n  bot_token: file\n  chat_id: \"1\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TG_BOT_TOKEN", "from-env")
	t.Setenv("TG_CHAT_ID", "99")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TGBotToken != "from-env" || cfg.TGChatID != "99" {
		t.Fatalf("got %q %q", cfg.TGBotToken, cfg.TGChatID)
	}
}

func TestLoadMissing(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadExample(t *testing.T) {
	t.Setenv("TG_BOT_TOKEN", "")
	t.Setenv("TG_CHAT_ID", "")
	cfg, err := Load(filepath.Join("..", "..", "configs", "config.example.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServerName != "my-vps" || cfg.NginxEnabled != "auto" {
		t.Fatalf("%+v", cfg)
	}
}
