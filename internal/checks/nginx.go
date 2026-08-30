package checks

import (
	"context"
	"strings"

	"github.com/voronkovd/gamayun/internal/config"
)

type Nginx struct {
	Cfg  config.Config
	Exec Exec
}

func (n Nginx) Run(ctx context.Context) []Result {
	decision := nginxDecision(ctx, n.Cfg, n.Exec)
	if decision.Skip {
		return []Result{skipped("nginx.active", decision.Reason)}
	}
	if decision.Err != "" && n.Cfg.NginxForced() {
		return []Result{fail("nginx.active", "nginx: "+decision.Err, map[string]string{"state": "error"})}
	}
	out, _ := n.Exec.run(ctx, "systemctl", "is-active", "nginx")
	state := strings.TrimSpace(out)
	if state == "" {
		state = "unknown"
	}
	metrics := map[string]string{"state": state}
	if state == "active" {
		return []Result{ok("nginx.active", "nginx: active", metrics)}
	}
	return []Result{fail("nginx.active", "nginx: NOT active (state: "+state+")", metrics)}
}

type nginxMode struct {
	Skip   bool
	Reason string
	Err    string
}

func nginxDecision(ctx context.Context, cfg config.Config, exec Exec) nginxMode {
	if cfg.NginxDisabled() {
		return nginxMode{Skip: true, Reason: "nginx disabled"}
	}
	_, err := exec.lookPath("systemctl")
	if err != nil {
		if cfg.NginxForced() {
			return nginxMode{Err: "systemctl not found"}
		}
		return nginxMode{Skip: true, Reason: "systemctl not found"}
	}
	out, runErr := exec.run(ctx, "systemctl", "show", "nginx.service", "-p", "LoadState")
	load := parseLoadState(out)
	if runErr != nil && load == "" {
		if cfg.NginxForced() {
			return nginxMode{Err: "cannot query nginx unit"}
		}
		return nginxMode{Skip: true, Reason: "cannot query nginx unit"}
	}
	if load == "not-found" {
		if cfg.NginxForced() {
			return nginxMode{}
		}
		return nginxMode{Skip: true, Reason: "nginx unit not found"}
	}
	return nginxMode{}
}

func parseLoadState(out string) string {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "LoadState=") {
			return strings.TrimPrefix(line, "LoadState=")
		}
		if line != "" && !strings.Contains(line, "=") {
			return line
		}
	}
	return ""
}
