package service

import (
	"context"
	"strings"
	"testing"

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
		fakeCheck{[]checks.Result{{Key: "disk.root", OK: true, Message: "disk /: 10% used"}}},
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

type fakeCheck struct{ out []checks.Result }

func (f fakeCheck) Run(ctx context.Context) []checks.Result { return f.out }
