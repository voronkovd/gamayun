package alert

import (
	"fmt"
	"time"

	"github.com/voronkovd/gamayun/internal/checks"
	"github.com/voronkovd/gamayun/internal/config"
	"github.com/voronkovd/gamayun/internal/state"
)

const (
	KindProblem   = "PROBLEM"
	KindRemind    = "REMIND"
	KindRecovered = "RECOVERED"
)

type Event struct {
	Kind    string
	Key     string
	Message string
	Text    string
}

type FSM struct {
	FailStreak    int
	RecoverStreak int
	Escalation    []time.Duration
	ServerName    string
	Now           func() time.Time
}

func New(cfg config.Config) *FSM {
	return &FSM{
		FailStreak:    cfg.FailStreak,
		RecoverStreak: cfg.RecoverStreak,
		Escalation:    cfg.Escalation,
		ServerName:    cfg.ServerName,
		Now:           time.Now,
	}
}

func (f *FSM) Next(snap *state.File, results []checks.Result) (*state.File, []Event) {
	now := f.Now()
	out := state.Clone(snap)
	var events []Event

	for _, res := range results {
		if res.Skip {
			continue
		}
		ev, cs := f.step(out.Checks[res.Key], res, now)
		out.Checks[res.Key] = cs
		if ev == nil {
			f.touchOpenIncident(out, res.Key, res.Message)
			continue
		}
		events = append(events, *ev)
		switch ev.Kind {
		case KindProblem:
			out.Incidents = append(out.Incidents, state.Incident{
				Check:       res.Key,
				Started:     now,
				LastMessage: res.Message,
			})
		case KindRemind:
			if i := openIncident(out, res.Key); i >= 0 {
				out.Incidents[i].Reminders++
				out.Incidents[i].LastMessage = res.Message
			}
		case KindRecovered:
			if i := openIncident(out, res.Key); i >= 0 {
				t := now
				out.Incidents[i].Resolved = &t
				out.Incidents[i].LastMessage = res.Message
			}
		}
	}
	return out, events
}

func (f *FSM) step(cs state.CheckState, res checks.Result, now time.Time) (*Event, state.CheckState) {
	if !res.OK {
		cs.OKStreak = 0
		cs.FailStreak++
		cs.LastMessage = res.Message
		if cs.Status != "firing" {
			if cs.FailStreak >= f.FailStreak {
				cs.Status = "firing"
				cs.FirstSeen = now
				cs.LastAlert = now
				cs.AlertCount = 1
				cs.NextRemind = now.Add(f.escalationAt(0))
				ev := f.event(KindProblem, res.Key, res.Message, now, cs)
				return &ev, cs
			}
			return nil, cs
		}
		if !cs.NextRemind.IsZero() && !now.Before(cs.NextRemind) {
			delay := f.escalationAt(cs.AlertCount)
			cs.AlertCount++
			cs.LastAlert = now
			cs.NextRemind = now.Add(delay)
			ev := f.event(KindRemind, res.Key, res.Message, now, cs)
			return &ev, cs
		}
		return nil, cs
	}

	cs.FailStreak = 0
	cs.OKStreak++
	if cs.Status == "firing" && cs.OKStreak >= f.RecoverStreak {
		ev := f.event(KindRecovered, res.Key, res.Message, now, cs)
		cs = state.CheckState{Status: "ok"}
		return &ev, cs
	}
	if cs.Status != "firing" {
		cs.Status = "ok"
		cs.LastMessage = ""
	}
	return nil, cs
}

func (f *FSM) escalationAt(i int) time.Duration {
	if len(f.Escalation) == 0 {
		return 5 * time.Minute
	}
	if i >= len(f.Escalation) {
		return f.Escalation[len(f.Escalation)-1]
	}
	return f.Escalation[i]
}

func (f *FSM) event(kind, key, message string, now time.Time, cs state.CheckState) Event {
	stamp := now.Format("2006-01-02 15:04:05 MST")
	var text string
	switch kind {
	case KindProblem:
		text = fmt.Sprintf("PROBLEM from %s — %s\n- %s", f.ServerName, stamp, message)
	case KindRemind:
		n := cs.AlertCount - 1
		if n < 1 {
			n = 1
		}
		text = fmt.Sprintf("REMIND #%d from %s — %s (open %s)\n- %s", n, f.ServerName, stamp, formatDur(now.Sub(cs.FirstSeen)), message)
	case KindRecovered:
		text = fmt.Sprintf("RECOVERED from %s — %s (lasted %s)\n- %s", f.ServerName, stamp, formatDur(now.Sub(cs.FirstSeen)), message)
	}
	return Event{Kind: kind, Key: key, Message: message, Text: text}
}

func formatDur(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	d = d.Round(time.Second)
	if d < time.Minute {
		return d.String()
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func openIncident(f *state.File, key string) int {
	for i := len(f.Incidents) - 1; i >= 0; i-- {
		if f.Incidents[i].Check == key && f.Incidents[i].Resolved == nil {
			return i
		}
	}
	return -1
}

func (f *FSM) touchOpenIncident(snap *state.File, key, message string) {
	if i := openIncident(snap, key); i >= 0 && message != "" {
		snap.Incidents[i].LastMessage = message
	}
	cs := snap.Checks[key]
	if cs.Status == "firing" && message != "" {
		cs.LastMessage = message
		snap.Checks[key] = cs
	}
}
