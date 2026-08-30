package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const DefaultPath = "/etc/gamayun/config.yaml"

type Config struct {
	ServerName    string
	TGBotToken    string
	TGChatID      string
	DiskPctMax    int
	CertDaysMin   int
	MemAvailMinMB int
	Load15Max     float64
	Containers    []string
	CheckInterval time.Duration
	FailStreak    int
	RecoverStreak int
	Escalation    []time.Duration
	DigestHour    int
	DigestMin     int
	NginxEnabled  string
	StatePath     string
	ProcFS        string
	LetsEncrypt   string
	ConfigPath    string
	GitHubRepo    string
}

type fileConfig struct {
	ServerName string `yaml:"server_name"`
	Telegram   struct {
		BotToken string     `yaml:"bot_token"`
		ChatID   yamlString `yaml:"chat_id"`
	} `yaml:"telegram"`
	Checks struct {
		Interval      string   `yaml:"interval"`
		Nginx         string   `yaml:"nginx"`
		DiskPctMax    *int     `yaml:"disk_pct_max"`
		CertDaysMin   *int     `yaml:"cert_days_min"`
		MemAvailMinMB *int     `yaml:"mem_avail_min_mb"`
		Load15Max     *float64 `yaml:"load15_max"`
		Containers    []string `yaml:"containers"`
	} `yaml:"checks"`
	Alerts struct {
		FailStreak    *int     `yaml:"fail_streak"`
		RecoverStreak *int     `yaml:"recover_streak"`
		Escalation    []string `yaml:"escalation"`
	} `yaml:"alerts"`
	Digest struct {
		At string `yaml:"at"`
	} `yaml:"digest"`
	Paths struct {
		State string `yaml:"state"`
	} `yaml:"paths"`
	Update struct {
		GitHubRepo string `yaml:"github_repo"`
	} `yaml:"update"`
}

// yamlString accepts both 123456789 and "123456789" in YAML.
type yamlString string

func (s *yamlString) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.ScalarNode {
		return fmt.Errorf("want a string or number")
	}
	*s = yamlString(value.Value)
	return nil
}

func Defaults() Config {
	host, _ := os.Hostname()
	return Config{
		ServerName:    host,
		DiskPctMax:    85,
		CertDaysMin:   14,
		MemAvailMinMB: 150,
		Load15Max:     2.0,
		CheckInterval: time.Minute,
		FailStreak:    2,
		RecoverStreak: 1,
		Escalation:    []time.Duration{5 * time.Minute, 30 * time.Minute, 2 * time.Hour},
		DigestHour:    8,
		DigestMin:     0,
		NginxEnabled:  "auto",
		StatePath:     "/var/lib/gamayun/state.json",
		ProcFS:        "/proc",
		LetsEncrypt:   "/etc/letsencrypt/live",
		ConfigPath:    DefaultPath,
	}
}

func Load(path string) (Config, error) {
	if path == "" {
		path = DefaultPath
	}
	cfg := Defaults()
	cfg.ConfigPath = path

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config %s: %w", path, err)
	}
	var file fileConfig
	if err := yaml.Unmarshal(data, &file); err != nil {
		return cfg, fmt.Errorf("parse config %s: %w", path, err)
	}
	if err := applyFile(&cfg, file); err != nil {
		return cfg, err
	}
	applyEnvOverrides(&cfg)
	if cfg.FailStreak < 1 {
		cfg.FailStreak = 1
	}
	if cfg.RecoverStreak < 1 {
		cfg.RecoverStreak = 1
	}
	if cfg.CheckInterval <= 0 {
		return cfg, fmt.Errorf("checks.interval must be > 0")
	}
	return cfg, nil
}

func applyFile(cfg *Config, file fileConfig) error {
	if file.ServerName != "" {
		cfg.ServerName = file.ServerName
	}
	if file.Telegram.BotToken != "" {
		cfg.TGBotToken = file.Telegram.BotToken
	}
	if file.Telegram.ChatID != "" {
		cfg.TGChatID = string(file.Telegram.ChatID)
	}
	if file.Checks.Interval != "" {
		d, err := time.ParseDuration(file.Checks.Interval)
		if err != nil {
			return fmt.Errorf("checks.interval: %w", err)
		}
		cfg.CheckInterval = d
	}
	if file.Checks.Nginx != "" {
		cfg.NginxEnabled = strings.ToLower(file.Checks.Nginx)
	}
	if file.Checks.DiskPctMax != nil {
		cfg.DiskPctMax = *file.Checks.DiskPctMax
	}
	if file.Checks.CertDaysMin != nil {
		cfg.CertDaysMin = *file.Checks.CertDaysMin
	}
	if file.Checks.MemAvailMinMB != nil {
		cfg.MemAvailMinMB = *file.Checks.MemAvailMinMB
	}
	if file.Checks.Load15Max != nil {
		cfg.Load15Max = *file.Checks.Load15Max
	}
	if file.Checks.Containers != nil {
		cfg.Containers = file.Checks.Containers
	}
	if file.Alerts.FailStreak != nil {
		cfg.FailStreak = *file.Alerts.FailStreak
	}
	if file.Alerts.RecoverStreak != nil {
		cfg.RecoverStreak = *file.Alerts.RecoverStreak
	}
	if len(file.Alerts.Escalation) > 0 {
		esc := make([]time.Duration, 0, len(file.Alerts.Escalation))
		for _, p := range file.Alerts.Escalation {
			d, err := time.ParseDuration(p)
			if err != nil {
				return fmt.Errorf("alerts.escalation: %w", err)
			}
			esc = append(esc, d)
		}
		cfg.Escalation = esc
	}
	if file.Digest.At != "" {
		h, m, err := ParseClock(file.Digest.At)
		if err != nil {
			return fmt.Errorf("digest.at: %w", err)
		}
		cfg.DigestHour, cfg.DigestMin = h, m
	}
	if file.Paths.State != "" {
		cfg.StatePath = file.Paths.State
	}
	if file.Update.GitHubRepo != "" {
		cfg.GitHubRepo = file.Update.GitHubRepo
	}
	return nil
}

func applyEnvOverrides(cfg *Config) {
	if v := env("TG_BOT_TOKEN", "VPS_TG_BOT_TOKEN"); v != "" {
		cfg.TGBotToken = v
	}
	if v := env("TG_CHAT_ID", "VPS_TG_CHAT_ID"); v != "" {
		cfg.TGChatID = v
	}
	if v := env("GAMAYUN_REPO"); v != "" {
		cfg.GitHubRepo = v
	}
}

func env(keys ...string) string {
	for _, key := range keys {
		if v, ok := os.LookupEnv(key); ok && v != "" {
			return v
		}
	}
	return ""
}

func ParseClock(v string) (hour, min int, err error) {
	parts := strings.Split(strings.TrimSpace(v), ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("want HH:MM")
	}
	hour, err = strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf("invalid hour")
	}
	min, err = strconv.Atoi(parts[1])
	if err != nil || min < 0 || min > 59 {
		return 0, 0, fmt.Errorf("invalid minute")
	}
	return hour, min, nil
}

func (c Config) TelegramOK() bool {
	return c.TGBotToken != "" && c.TGChatID != ""
}

func (c Config) NginxForced() bool {
	switch c.NginxEnabled {
	case "1", "true", "on", "yes":
		return true
	default:
		return false
	}
}

func (c Config) NginxDisabled() bool {
	switch c.NginxEnabled {
	case "0", "false", "off", "no":
		return true
	default:
		return false
	}
}
