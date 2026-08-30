package digest

import (
	"fmt"
	"strings"
	"time"

	"github.com/voronkovd/gamayun/internal/checks"
	"github.com/voronkovd/gamayun/internal/state"
)

func TodaySlot(now time.Time, hour, min int) time.Time {
	return time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, now.Location())
}

func LastSlot(now time.Time, hour, min int) time.Time {
	slot := TodaySlot(now, hour, min)
	if now.Before(slot) {
		return slot.Add(-24 * time.Hour)
	}
	return slot
}

func ShouldSend(now, lastDigest time.Time, hour, min int) bool {
	today := TodaySlot(now, hour, min)
	if lastDigest.IsZero() {
		return !now.Before(today)
	}
	return lastDigest.Before(LastSlot(now, hour, min))
}

func Relevant(incidents []state.Incident, lastDigest, now time.Time) []state.Incident {
	since := lastDigest
	if since.IsZero() {
		since = now.Add(-24 * time.Hour)
	}
	var out []state.Incident
	for _, inc := range incidents {
		if !inc.Started.Before(since) {
			out = append(out, inc)
			continue
		}
		if inc.Resolved == nil || inc.Resolved.After(since) {
			out = append(out, inc)
		}
	}
	return out
}

func Prune(incidents []state.Incident, now time.Time) []state.Incident {
	cutoff := now.Add(-7 * 24 * time.Hour)
	var out []state.Incident
	for _, inc := range incidents {
		if inc.Resolved == nil || inc.Resolved.After(cutoff) {
			out = append(out, inc)
		}
	}
	return out
}

func Format(server string, now time.Time, results []checks.Result, incidents []state.Incident) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s daily %s\n\n", server, now.Format("2006-01-02"))
	b.WriteString("Snapshot:\n")
	if len(results) == 0 {
		b.WriteString("- (no checks)\n")
	}
	for _, r := range results {
		b.WriteString("- ")
		b.WriteString(snapshotLine(r))
		b.WriteByte('\n')
	}
	b.WriteString("\nIncidents:\n")
	if len(incidents) == 0 {
		b.WriteString("- none\n")
		return b.String()
	}
	for _, inc := range incidents {
		b.WriteString("- ")
		b.WriteString(incidentLine(inc, now))
		b.WriteByte('\n')
	}
	return b.String()
}

func snapshotLine(r checks.Result) string {
	if r.Skip {
		if r.Message != "" {
			return r.Key + ": skipped (" + r.Message + ")"
		}
		return r.Key + ": skipped"
	}
	if r.Message != "" {
		return r.Message
	}
	if r.OK {
		return r.Key + ": ok"
	}
	return r.Key + ": FAIL"
}

func incidentLine(inc state.Incident, now time.Time) string {
	start := inc.Started.Format("15:04")
	end := "OPEN"
	if inc.Resolved != nil {
		end = inc.Resolved.Format("15:04")
	} else {
		_ = now
	}
	msg := inc.LastMessage
	if msg == "" {
		msg = inc.Check
	}
	return fmt.Sprintf("%s: %s–%s, %d reminds, %s", inc.Check, start, end, inc.Reminders, msg)
}
