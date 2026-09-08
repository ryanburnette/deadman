// Package state persists each service's last-known heartbeat so a restart
// doesn't forget history mid-outage. It's a plain JSON text file.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Status string

const (
	Up   Status = "up"
	Down Status = "down"
)

// Service is the runtime state tracked for one monitored service.
type Service struct {
	LastSeen     time.Time `json:"last_seen"`
	Status       Status    `json:"status"`
	LastNotified time.Time `json:"last_notified"`
}

// State maps service name to its runtime state.
type State map[string]Service

// Load reads state.json. A missing file is treated as empty state.
func Load(path string) (State, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return State{}, nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	if len(b) == 0 {
		return State{}, nil
	}
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return s, nil
}

// Save writes state.json atomically (write to a temp file, then rename).
func Save(path string, s State) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding %s: %w", path, err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("renaming %s to %s: %w", tmp, path, err)
	}
	return nil
}
