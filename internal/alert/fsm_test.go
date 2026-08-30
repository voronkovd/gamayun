package alert

import (
	"testing"
	"time"

	"github.com/voronkovd/gamayun/internal/checks"
	"github.com/voronkovd/gamayun/internal/state"
)

func testFSM(now *time.Time) *FSM {
	return &FSM{
		FailStreak:    2,
		RecoverStreak: 1,
		Escalation:    []time.Duration{5 * time.Minute, 30 * time.Minute, 2 * time.Hour},
		ServerName:    "lab",
		Now:           func() time.Time { return *now },
	}
}

func failRes(msg string) checks.Result {
	return checks.Result{Key: "disk.root", OK: false, Message: msg}
}

func okRes(msg string) checks.Result {
	return checks.Result{Key: "disk.root", OK: true, Message: msg}
}

func TestFlapDoesNotAlert(t *testing.T) {
	now := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	f := testFSM(&now)
	snap := state.Empty()

	snap, ev := f.Next(snap, []checks.Result{failRes("disk /: 90% used (>=85%)")})
	if len(ev) != 0 {
		t.Fatalf("first fail must not alert: %v", ev)
	}
	snap, ev = f.Next(snap, []checks.Result{okRes("disk /: 40% used")})
	if len(ev) != 0 {
		t.Fatalf("ok after one fail: %v", ev)
	}
	if snap.Checks["disk.root"].Status == "firing" {
		t.Fatal("should not be firing")
	}
}

func TestProblemRemindRecover(t *testing.T) {
	now := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	f := testFSM(&now)
	snap := state.Empty()
	msg := "disk /: 90% used (>=85%)"

	snap, ev := f.Next(snap, []checks.Result{failRes(msg)})
	if len(ev) != 0 {
		t.Fatal(ev)
	}
	now = now.Add(time.Minute)
	snap, ev = f.Next(snap, []checks.Result{failRes(msg)})
	if len(ev) != 1 || ev[0].Kind != KindProblem {
		t.Fatalf("want PROBLEM, got %+v", ev)
	}
	if snap.Checks["disk.root"].Status != "firing" {
		t.Fatal(snap.Checks["disk.root"])
	}
	if openIncident(snap, "disk.root") < 0 {
		t.Fatal("incident not opened")
	}

	now = now.Add(4 * time.Minute)
	snap, ev = f.Next(snap, []checks.Result{failRes(msg)})
	if len(ev) != 0 {
		t.Fatalf("too early for remind: %v", ev)
	}

	now = now.Add(time.Minute)
	snap, ev = f.Next(snap, []checks.Result{failRes(msg)})
	if len(ev) != 1 || ev[0].Kind != KindRemind {
		t.Fatalf("want REMIND, got %+v", ev)
	}
	if snap.Incidents[0].Reminders != 1 {
		t.Fatalf("reminders=%d", snap.Incidents[0].Reminders)
	}

	now = now.Add(30 * time.Minute)
	snap, ev = f.Next(snap, []checks.Result{failRes("disk /: 95% used (>=85%)")})
	if len(ev) != 1 || ev[0].Kind != KindRemind {
		t.Fatalf("second remind: %+v", ev)
	}

	now = now.Add(time.Minute)
	snap, ev = f.Next(snap, []checks.Result{okRes("disk /: 40% used")})
	if len(ev) != 1 || ev[0].Kind != KindRecovered {
		t.Fatalf("want RECOVERED, got %+v", ev)
	}
	if snap.Checks["disk.root"].Status != "ok" {
		t.Fatal(snap.Checks["disk.root"])
	}
	if snap.Incidents[0].Resolved == nil {
		t.Fatal("incident not closed")
	}
}

func TestMessageChangeIsNotNewProblem(t *testing.T) {
	now := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	f := testFSM(&now)
	snap := state.Empty()

	snap, _ = f.Next(snap, []checks.Result{{Key: "docker.unhealthy", OK: false, Message: "docker unhealthy: a"}})
	now = now.Add(time.Minute)
	snap, ev := f.Next(snap, []checks.Result{{Key: "docker.unhealthy", OK: false, Message: "docker unhealthy: a"}})
	if len(ev) != 1 || ev[0].Kind != KindProblem {
		t.Fatalf("got %+v", ev)
	}
	now = now.Add(time.Minute)
	snap, ev = f.Next(snap, []checks.Result{{Key: "docker.unhealthy", OK: false, Message: "docker unhealthy: b"}})
	if len(ev) != 0 {
		t.Fatalf("text change must wait for remind: %+v", ev)
	}
	if snap.Incidents[0].LastMessage != "docker unhealthy: b" {
		t.Fatalf("message not updated: %s", snap.Incidents[0].LastMessage)
	}
}

func TestSkipIgnored(t *testing.T) {
	now := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	f := testFSM(&now)
	snap := state.Empty()
	snap, ev := f.Next(snap, []checks.Result{{Key: "nginx.active", Skip: true, OK: true, Message: "disabled"}})
	if len(ev) != 0 || len(snap.Checks) != 0 {
		t.Fatalf("skip should be ignored: ev=%v checks=%v", ev, snap.Checks)
	}
}
