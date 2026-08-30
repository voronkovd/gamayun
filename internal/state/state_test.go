package state

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLoadSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "state.json")
	in := Empty()
	in.Checks["disk.root"] = CheckState{Status: "firing", FailStreak: 2, LastMessage: "full"}
	now := time.Date(2026, 8, 30, 8, 0, 0, 0, time.UTC)
	in.LastDigest = now
	in.Incidents = []Incident{{Check: "disk.root", Started: now, LastMessage: "full"}}
	if err := Save(path, in); err != nil {
		t.Fatal(err)
	}
	out, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if out.Checks["disk.root"].Status != "firing" || out.LastDigest != now {
		t.Fatalf("%+v", out)
	}
	if len(out.Incidents) != 1 {
		t.Fatal(out.Incidents)
	}
}

func TestLoadMissing(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil || got == nil || got.Checks == nil {
		t.Fatalf("%v %+v", err, got)
	}
}
