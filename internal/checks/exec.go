package checks

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

type Exec struct {
	LookPath func(file string) (string, error)
	Run      func(ctx context.Context, name string, args ...string) (string, error)
	Timeout  time.Duration
}

func DefaultExec() Exec {
	return Exec{
		LookPath: exec.LookPath,
		Run: func(ctx context.Context, name string, args ...string) (string, error) {
			cmd := exec.CommandContext(ctx, name, args...)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err := cmd.Run()
			out := strings.TrimSpace(stdout.String())
			if err != nil && out == "" {
				out = strings.TrimSpace(stderr.String())
			}
			return out, err
		},
		Timeout: 15 * time.Second,
	}
}

func (e Exec) lookPath(name string) (string, error) {
	if e.LookPath != nil {
		return e.LookPath(name)
	}
	return exec.LookPath(name)
}

func (e Exec) run(ctx context.Context, name string, args ...string) (string, error) {
	if e.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.Timeout)
		defer cancel()
	}
	if e.Run != nil {
		return e.Run(ctx, name, args...)
	}
	return DefaultExec().Run(ctx, name, args...)
}
