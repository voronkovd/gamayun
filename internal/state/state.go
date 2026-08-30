package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type CheckState struct {
	Status      string    `json:"status"`
	FailStreak  int       `json:"fail_streak"`
	OKStreak    int       `json:"ok_streak"`
	FirstSeen   time.Time `json:"first_seen,omitempty"`
	LastAlert   time.Time `json:"last_alert,omitempty"`
	NextRemind  time.Time `json:"next_remind,omitempty"`
	AlertCount  int       `json:"alert_count"`
	LastMessage string    `json:"last_message,omitempty"`
}

type Incident struct {
	Check       string     `json:"check"`
	Started     time.Time  `json:"started"`
	Resolved    *time.Time `json:"resolved"`
	Reminders   int        `json:"reminders"`
	LastMessage string     `json:"last_message"`
}

type File struct {
	Checks     map[string]CheckState `json:"checks"`
	Incidents  []Incident            `json:"incidents"`
	LastDigest time.Time             `json:"last_digest,omitempty"`
}

func Empty() *File {
	return &File{Checks: map[string]CheckState{}}
}

func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Empty(), nil
		}
		return nil, err
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	if f.Checks == nil {
		f.Checks = map[string]CheckState{}
	}
	return &f, nil
}

func Save(path string, f *File) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func Clone(f *File) *File {
	out := &File{
		LastDigest: f.LastDigest,
		Checks:     make(map[string]CheckState, len(f.Checks)),
		Incidents:  append([]Incident(nil), f.Incidents...),
	}
	for k, v := range f.Checks {
		out.Checks[k] = v
	}
	return out
}
