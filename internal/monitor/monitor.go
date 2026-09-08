// Package monitor ties config, state, and mailer together: it watches
// services.csv for edits, tracks each service's heartbeat deadline, and
// fires the down/recovery emails.
package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/ryanburnette/deadman/internal/config"
	"github.com/ryanburnette/deadman/internal/mailer"
	"github.com/ryanburnette/deadman/internal/state"
)

// Sender is the subset of *mailer.Mailer the monitor needs, so it can be
// swapped out in tests.
type Sender interface {
	Send(to, subject, body string) error
}

type Monitor struct {
	configPath string
	statePath  string
	mailer     Sender

	mu            sync.Mutex
	services      map[string]config.Service
	st            state.State
	configModTime time.Time
}

// New loads the initial config and state from disk.
func New(configPath, statePath string, sender Sender) (*Monitor, error) {
	m := &Monitor{
		configPath: configPath,
		statePath:  statePath,
		mailer:     sender,
	}

	st, err := state.Load(statePath)
	if err != nil {
		return nil, err
	}
	m.st = st

	if err := m.reloadConfig(); err != nil {
		return nil, err
	}
	return m, nil
}

// reloadConfig re-reads services.csv and reconciles state: new services get
// a fresh grace period starting now, removed services drop their state.
// Caller must hold the lock.
func (m *Monitor) reloadConfig() error {
	services, err := config.Load(m.configPath)
	if err != nil {
		return err
	}

	byName := make(map[string]config.Service, len(services))
	for _, s := range services {
		byName[s.Name] = s
		if _, ok := m.st[s.Name]; !ok {
			if m.st == nil {
				m.st = state.State{}
			}
			m.st[s.Name] = state.Service{LastSeen: now(), Status: state.Up}
		}
	}
	for name := range m.st {
		if _, ok := byName[name]; !ok {
			delete(m.st, name)
		}
	}

	m.services = byName
	if fi, err := os.Stat(m.configPath); err == nil {
		m.configModTime = fi.ModTime()
	}
	return state.Save(m.statePath, m.st)
}

func (m *Monitor) maybeReloadConfig() {
	fi, err := os.Stat(m.configPath)
	if err != nil {
		return
	}
	if fi.ModTime().Equal(m.configModTime) {
		return
	}
	if err := m.reloadConfig(); err != nil {
		slog.Error("reloading config", "error", err)
		return
	}
	slog.Info("config reloaded", "services", len(m.services))
}

// Run polls for config changes and missed deadlines every checkInterval,
// until ctx is canceled.
func (m *Monitor) Run(ctx context.Context, checkInterval time.Duration) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.tick()
		}
	}
}

func (m *Monitor) tick() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.maybeReloadConfig()

	dirty := false
	for name, svc := range m.services {
		st := m.st[name]
		deadline := st.LastSeen.Add(svc.Deadline())
		n := now()
		if !n.After(deadline) {
			continue
		}

		switch st.Status {
		case state.Up:
			m.alertDown(svc, st, n)
			st.Status = state.Down
			st.LastNotified = n
			m.st[name] = st
			dirty = true
		case state.Down:
			if svc.Repeat > 0 && n.Sub(st.LastNotified) >= svc.Repeat {
				m.alertDown(svc, st, n)
				st.LastNotified = n
				m.st[name] = st
				dirty = true
			}
		}
	}

	if dirty {
		if err := state.Save(m.statePath, m.st); err != nil {
			slog.Error("saving state", "error", err)
		}
	}
}

func (m *Monitor) alertDown(svc config.Service, st state.Service, at time.Time) {
	subject, body := mailer.DownEmail(svc.Name, st.LastSeen, svc.Interval, svc.Grace, svc.Repeat, at.Sub(st.LastSeen))
	if err := m.mailer.Send(svc.Email, subject, body); err != nil {
		slog.Error("sending down alert", "service", svc.Name, "error", err)
		return
	}
	slog.Info("sent down alert", "service", svc.Name)
}

// CheckIn records a heartbeat for the named service if token matches,
// sending a recovery email if it had been marked down.
func (m *Monitor) CheckIn(name, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	svc, ok := m.services[name]
	if !ok || svc.Token != token {
		return fmt.Errorf("unknown service or bad token")
	}

	n := now()
	st := m.st[name]
	if st.Status == state.Down {
		downtime := n.Sub(st.LastSeen)
		subject, body := mailer.UpEmail(name, downtime)
		if err := m.mailer.Send(svc.Email, subject, body); err != nil {
			slog.Error("sending recovery notice", "service", name, "error", err)
		} else {
			slog.Info("sent recovery notice", "service", name)
		}
	}

	m.st[name] = state.Service{LastSeen: n, Status: state.Up}
	return state.Save(m.statePath, m.st)
}

var now = time.Now
