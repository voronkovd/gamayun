package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/voronkovd/gamayun/internal/alert"
	"github.com/voronkovd/gamayun/internal/checks"
	"github.com/voronkovd/gamayun/internal/config"
	"github.com/voronkovd/gamayun/internal/digest"
	"github.com/voronkovd/gamayun/internal/notify"
	"github.com/voronkovd/gamayun/internal/state"
)

type Daemon struct {
	Cfg    config.Config
	Runner *checks.Runner
	FSM    *alert.FSM
	Sender notify.Sender
	Now    func() time.Time
	Log    *log.Logger
	state  *state.File
}

func New(cfg config.Config, runner *checks.Runner, sender notify.Sender) (*Daemon, error) {
	snap, err := state.Load(cfg.StatePath)
	if err != nil {
		return nil, err
	}
	return &Daemon{
		Cfg:    cfg,
		Runner: runner,
		FSM:    alert.New(cfg),
		Sender: sender,
		Now:    time.Now,
		Log:    log.Default(),
		state:  snap,
	}, nil
}

func (d *Daemon) Run(ctx context.Context) error {
	if !d.Cfg.TelegramOK() {
		d.logf("telegram not configured: set telegram.bot_token and telegram.chat_id in %s", d.Cfg.ConfigPath)
	}
	if err := d.maybeDigest(ctx, false); err != nil {
		d.logf("digest on start: %v", err)
	}
	d.tick(ctx)

	ticker := time.NewTicker(d.Cfg.CheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			d.tick(ctx)
		}
	}
}

func (d *Daemon) tick(ctx context.Context) {
	results := d.Runner.Run(ctx)
	next, events := d.FSM.Next(d.state, results)
	for _, ev := range events {
		d.logf("%s %s: %s", ev.Kind, ev.Key, ev.Message)
		if err := d.send(ctx, ev.Text); err != nil {
			d.logf("%v", err)
		}
	}
	d.state = next
	if err := state.Save(d.Cfg.StatePath, d.state); err != nil {
		d.logf("state save: %v", err)
	}
	if err := d.maybeDigest(ctx, false); err != nil {
		d.logf("digest: %v", err)
	}
}

func (d *Daemon) maybeDigest(ctx context.Context, force bool) error {
	now := d.now()
	if !force && !digest.ShouldSend(now, d.state.LastDigest, d.Cfg.DigestHour, d.Cfg.DigestMin) {
		return nil
	}
	results := d.Runner.Run(ctx)
	return d.sendDigest(ctx, now, results)
}

func (d *Daemon) ForceDigest(ctx context.Context) error {
	return d.maybeDigest(ctx, true)
}

func (d *Daemon) sendDigest(ctx context.Context, now time.Time, results []checks.Result) error {
	incs := digest.Relevant(d.state.Incidents, d.state.LastDigest, now)
	text := digest.Format(d.Cfg.ServerName, now, results, incs)
	if err := d.send(ctx, strings.TrimRight(text, "\n")); err != nil {
		return err
	}
	d.state.LastDigest = now
	d.state.Incidents = digest.Prune(d.state.Incidents, now)
	if err := state.Save(d.Cfg.StatePath, d.state); err != nil {
		return fmt.Errorf("state save: %w", err)
	}
	d.logf("digest sent")
	return nil
}

func (d *Daemon) send(ctx context.Context, text string) error {
	if d.Sender == nil {
		return fmt.Errorf("telegram not configured: set telegram.bot_token and telegram.chat_id")
	}
	return d.Sender.Send(ctx, text)
}

func (d *Daemon) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now()
}

func (d *Daemon) logf(format string, args ...any) {
	if d.Log != nil {
		d.Log.Printf(format, args...)
	}
}

func PrintOnce(results []checks.Result) int {
	code := 0
	for _, r := range results {
		switch {
		case r.Skip:
			fmt.Printf("SKIP %s: %s\n", r.Key, r.Message)
		case r.OK:
			fmt.Printf("OK   %s: %s\n", r.Key, r.Message)
		default:
			fmt.Printf("FAIL %s: %s\n", r.Key, r.Message)
			code = 1
		}
	}
	if code == 0 {
		fmt.Println("OK: all checks passed")
	}
	return code
}
