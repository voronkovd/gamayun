package checks

import (
	"strings"
	"testing"
	"time"
)

func TestUsagePct(t *testing.T) {
	// 1000 blocks, 100 free to root, 50 avail to users → used=900, den=950 → 94%
	if got := usagePct(1000, 100, 50); got != 94 {
		t.Fatalf("got %d", got)
	}
	if got := usagePct(100, 100, 0); got != 0 {
		t.Fatalf("empty: %d", got)
	}
}

func TestParseMemAvailable(t *testing.T) {
	in := "MemTotal: 8000000 kB\nMemAvailable: 204800 kB\n"
	mb, err := parseMemAvailable(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if mb != 200 {
		t.Fatalf("got %d", mb)
	}
}

func TestParseLoad15(t *testing.T) {
	v, err := parseLoad15("0.12 0.34 1.5 1/234 99\n")
	if err != nil || v != 1.5 {
		t.Fatalf("got %v %v", v, err)
	}
}

func TestParseTCPLine(t *testing.T) {
	line := "0: 00000000:0050 00000000:0000 0A 00000000:00000000 00:00000000 00000000 0 0 1"
	port, listen, ok := parseTCPLine(line)
	if !ok || !listen || port != 80 {
		t.Fatalf("got port=%d listen=%v ok=%v", port, listen, ok)
	}
	line443 := "1: 00000000:01BB 00000000:0000 0A 00000000:00000000 00:00000000 00000000 0 0 2"
	port, listen, ok = parseTCPLine(line443)
	if !ok || !listen || port != 443 {
		t.Fatalf("443: port=%d listen=%v ok=%v", port, listen, ok)
	}
	established := "2: 00000000:0050 0100007F:1234 01 00000000:00000000 00:00000000 00000000 0 0 3"
	_, listen, ok = parseTCPLine(established)
	if !ok || listen {
		t.Fatal("established should not count as listen")
	}
}

func TestParseTCPTable(t *testing.T) {
	raw := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 00000000:0050 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 1
   1: 0100007F:1F90 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 2
`
	into := map[int]bool{}
	if err := parseTCPTable(strings.NewReader(raw), into); err != nil {
		t.Fatal(err)
	}
	if !into[80] || !into[8080] {
		t.Fatalf("got %v", into)
	}
}

func TestDaysUntil(t *testing.T) {
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	exp := now.Add(10 * 24 * time.Hour)
	if d := daysUntil(exp, now); d != 10 {
		t.Fatalf("got %d", d)
	}
}

func TestParseLoadState(t *testing.T) {
	if got := parseLoadState("LoadState=not-found"); got != "not-found" {
		t.Fatalf("got %q", got)
	}
	if got := parseLoadState("loaded"); got != "loaded" {
		t.Fatalf("got %q", got)
	}
}

func TestBadDockerStates(t *testing.T) {
	in := "web running\njob exited\nloop restarting\n"
	got := badDockerStates(in)
	if len(got) != 2 || got[0] != "job(exited)" || got[1] != "loop(restarting)" {
		t.Fatalf("got %v", got)
	}
}
