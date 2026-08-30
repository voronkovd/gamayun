package digest

import (
	"strings"
	"testing"
	"time"

	"github.com/voronkovd/gamayun/internal/checks"
	"github.com/voronkovd/gamayun/internal/state"
)

func TestShouldSend(t *testing.T) {
	loc := time.FixedZone("MSK", 3*3600)
	today0800 := time.Date(2026, 8, 30, 8, 0, 0, 0, loc)

	if ShouldSend(time.Date(2026, 8, 30, 7, 0, 0, 0, loc), time.Time{}, 8, 0) {
		t.Fatal("first run before slot must wait")
	}
	if !ShouldSend(time.Date(2026, 8, 30, 10, 0, 0, 0, loc), time.Time{}, 8, 0) {
		t.Fatal("first run after slot must send")
	}
	if ShouldSend(time.Date(2026, 8, 30, 10, 0, 0, 0, loc), today0800.Add(5*time.Second), 8, 0) {
		t.Fatal("already sent today")
	}
	if ShouldSend(time.Date(2026, 8, 31, 7, 0, 0, 0, loc), today0800, 8, 0) {
		t.Fatal("before next slot must wait")
	}
	if !ShouldSend(time.Date(2026, 8, 31, 8, 1, 0, 0, loc), today0800, 8, 0) {
		t.Fatal("next morning must send")
	}
	missed := time.Date(2026, 8, 29, 8, 0, 0, 0, loc)
	if !ShouldSend(time.Date(2026, 8, 30, 10, 0, 0, 0, loc), missed, 8, 0) {
		t.Fatal("Persistent: send missed slot on start after 08:00")
	}
}

func TestRelevantAndPrune(t *testing.T) {
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	last := time.Date(2026, 8, 30, 8, 0, 0, 0, time.UTC)
	resolved := time.Date(2026, 8, 30, 9, 0, 0, 0, time.UTC)
	oldResolved := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	incs := []state.Incident{
		{Check: "disk.root", Started: time.Date(2026, 8, 30, 9, 10, 0, 0, time.UTC), Resolved: &resolved, Reminders: 1, LastMessage: "disk"},
		{Check: "load.15", Started: time.Date(2026, 8, 30, 11, 0, 0, 0, time.UTC), LastMessage: "load"},
		{Check: "old", Started: time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC), Resolved: &oldResolved},
	}
	got := Relevant(incs, last, now)
	if len(got) != 2 {
		t.Fatalf("relevant=%d", len(got))
	}
	pruned := Prune(incs, now)
	if len(pruned) != 2 {
		t.Fatalf("prune=%d want 2 (drop 10-day-old)", len(pruned))
	}
}

func TestFormat(t *testing.T) {
	now := time.Date(2026, 8, 30, 8, 0, 0, 0, time.UTC)
	text := Format("lab", now, []checks.Result{
		{Key: "disk.root", OK: true, Message: "disk /: 42% used"},
		{Key: "nginx.active", Skip: true, Message: "nginx disabled"},
	}, nil)
	if !strings.Contains(text, "lab daily 2026-08-30") {
		t.Fatal(text)
	}
	if !strings.Contains(text, "disk /: 42% used") {
		t.Fatal(text)
	}
	if !strings.Contains(text, "skipped") {
		t.Fatal(text)
	}
	if !strings.Contains(text, "Incidents:\n- none") {
		t.Fatal(text)
	}
}
