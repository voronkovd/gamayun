package service

import (
	"context"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/voronkovd/gamayun/internal/checks"
	"github.com/voronkovd/gamayun/internal/config"
)

func TestPrintOnce(t *testing.T) {
	code := PrintOnce([]checks.Result{
		{Key: "disk.root", OK: true, Message: "ok"},
		{Key: "nginx.active", Skip: true, Message: "disabled"},
	})
	if code != 0 {
		t.Fatalf("code=%d", code)
	}
	code = PrintOnce([]checks.Result{{Key: "disk.root", OK: false, Message: "full"}})
	if code != 1 {
		t.Fatalf("code=%d", code)
	}
}

type recSender struct{ msgs []string }

func (r *recSender) Send(ctx context.Context, text string) error {
	r.msgs = append(r.msgs, text)
	return nil
}

func TestForceDigestUsesSender(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Defaults()
	cfg.ServerName = "lab"
	cfg.StatePath = dir + "/state.json"
	cfg.NginxEnabled = "0"
	cfg.ProcFS = dir
	sender := &recSender{}
	d, err := New(cfg, &checks.Runner{Checks: []checks.Check{
		&fakeCheck{out: []checks.Result{{Key: "disk.root", OK: true, Message: "disk /: 10% used"}}},
	}}, sender)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.ForceDigest(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(sender.msgs) != 1 || !strings.Contains(sender.msgs[0], "lab daily") {
		t.Fatalf("%v", sender.msgs)
	}
	if d.state.LastDigest.IsZero() {
		t.Fatal("last_digest not set")
	}
}

func TestDaemonRunLifecycle(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Defaults()
	cfg.ServerName = "lab"
	cfg.StatePath = dir + "/state.json"
	cfg.CheckInterval = 20 * time.Millisecond
	// Digest slot in the future relative to fixed Now — no digest on start or ticks.
	cfg.DigestHour = 23
	cfg.DigestMin = 0

	check := &fakeCheck{out: []checks.Result{
		{Key: "disk.root", OK: true, Message: "disk /: 10% used"},
	}}
	sender := &recSender{}
	d, err := New(cfg, &checks.Runner{Checks: []checks.Check{check}}, sender)
	if err != nil {
		t.Fatal(err)
	}
	fixed := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	d.Now = func() time.Time { return fixed }

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- d.Run(ctx) }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after context deadline")
	}

	if n := check.calls.Load(); n < 2 {
		t.Fatalf("Runner.Run called %d times, want >= 2", n)
	}
	info, err := os.Stat(cfg.StatePath)
	if err != nil {
		t.Fatalf("state file: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("state file is empty")
	}
}

type fakeCheck struct {
	out   []checks.Result
	calls atomic.Int32
}

func (f *fakeCheck) Run(ctx context.Context) []checks.Result {
	f.calls.Add(1)
	return f.out
}
