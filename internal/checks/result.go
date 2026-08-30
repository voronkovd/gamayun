package checks

import (
	"context"

	"github.com/voronkovd/gamayun/internal/config"
)

type Result struct {
	Key     string
	OK      bool
	Message string
	Metrics map[string]string
	Skip    bool
}

type Check interface {
	Run(ctx context.Context) []Result
}

type Runner struct {
	Checks []Check
}

func (r *Runner) Run(ctx context.Context) []Result {
	var out []Result
	for _, c := range r.Checks {
		out = append(out, c.Run(ctx)...)
	}
	return out
}

func Default(cfg config.Config, exec Exec) *Runner {
	if exec.LookPath == nil && exec.Run == nil {
		exec = DefaultExec()
	}
	return &Runner{Checks: []Check{
		NginxGroup{Cfg: cfg, Exec: exec},
		Certs{Cfg: cfg},
		Disk{Cfg: cfg},
		Memory{Cfg: cfg},
		Load{Cfg: cfg},
		Docker{Cfg: cfg, Exec: exec},
	}}
}

func skipped(key, reason string) Result {
	return Result{
		Key:     key,
		OK:      true,
		Skip:    true,
		Message: reason,
		Metrics: map[string]string{"status": "skipped"},
	}
}

func fail(key, msg string, metrics map[string]string) Result {
	return Result{Key: key, OK: false, Message: msg, Metrics: metrics}
}

func ok(key, msg string, metrics map[string]string) Result {
	return Result{Key: key, OK: true, Message: msg, Metrics: metrics}
}
