package monitor

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ryanburnette/deadman/internal/config"
	"github.com/ryanburnette/deadman/internal/state"
)

type fakeSender struct {
	mu   sync.Mutex
	sent []string // recipient of each send, in order
}

func (f *fakeSender) Send(to, subject, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, to)
	return nil
}

func (f *fakeSender) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.sent)
}

// withClock overrides the package clock for the duration of the test.
func withClock(t *testing.T, start time.Time) *time.Time {
	t.Helper()
	clock := start
	orig := now
	now = func() time.Time { return clock }
	t.Cleanup(func() { now = orig })
	return &clock
}

func TestTickMarksDownOnceByDefault(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "services.csv")
	statePath := filepath.Join(dir, "state.json")

	clock := withClock(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	if err := config.Save(configPath, []config.Service{
		{Name: "web1", Token: "tok1", Interval: time.Minute, Grace: 0, Repeat: 0, Email: "a@example.com"},
	}); err != nil {
		t.Fatalf("config.Save: %v", err)
	}

	sender := &fakeSender{}
	m, err := New(configPath, statePath, sender)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Still within the interval: no alert yet.
	*clock = clock.Add(30 * time.Second)
	m.tick()
	if got := sender.count(); got != 0 {
		t.Fatalf("expected no alert before the deadline, got %d sends", got)
	}

	// Past the deadline: exactly one alert.
	*clock = clock.Add(time.Minute)
	m.tick()
	if got := sender.count(); got != 1 {
		t.Fatalf("expected exactly one down alert, got %d", got)
	}

	// Still down, well past the deadline again, but repeat is 0: no more alerts.
	*clock = clock.Add(10 * time.Minute)
	m.tick()
	if got := sender.count(); got != 1 {
		t.Fatalf("expected repeat=0 to stay silent, got %d sends", got)
	}
}

func TestTickResendsOnRepeatInterval(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "services.csv")
	statePath := filepath.Join(dir, "state.json")

	clock := withClock(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	if err := config.Save(configPath, []config.Service{
		{Name: "web1", Token: "tok1", Interval: time.Minute, Grace: 0, Repeat: 5 * time.Minute, Email: "a@example.com"},
	}); err != nil {
		t.Fatalf("config.Save: %v", err)
	}

	sender := &fakeSender{}
	m, err := New(configPath, statePath, sender)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	*clock = clock.Add(time.Minute + time.Second)
	m.tick() // first down alert
	if got := sender.count(); got != 1 {
		t.Fatalf("expected 1 alert, got %d", got)
	}

	*clock = clock.Add(2 * time.Minute)
	m.tick() // still down, before repeat interval elapses
	if got := sender.count(); got != 1 {
		t.Fatalf("expected no resend before the repeat interval, got %d", got)
	}

	*clock = clock.Add(4 * time.Minute)
	m.tick() // now past the repeat interval since the last alert
	if got := sender.count(); got != 2 {
		t.Fatalf("expected a resend after the repeat interval, got %d", got)
	}
}

func TestCheckInRecoversAndRejectsBadToken(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "services.csv")
	statePath := filepath.Join(dir, "state.json")

	clock := withClock(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	if err := config.Save(configPath, []config.Service{
		{Name: "web1", Token: "tok1", Interval: time.Minute, Grace: 0, Repeat: 0, Email: "a@example.com"},
	}); err != nil {
		t.Fatalf("config.Save: %v", err)
	}

	sender := &fakeSender{}
	m, err := New(configPath, statePath, sender)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := m.CheckIn("web1", "wrong-token"); err == nil {
		t.Fatal("expected an error for a bad token")
	}
	if err := m.CheckIn("no-such-service", "tok1"); err == nil {
		t.Fatal("expected an error for an unknown service")
	}
	if got := sender.count(); got != 0 {
		t.Fatalf("bad check-ins should not send mail, got %d sends", got)
	}

	// Drive it down, then confirm a valid check-in recovers it and fires
	// exactly one recovery email.
	*clock = clock.Add(2 * time.Minute)
	m.tick()
	if got := sender.count(); got != 1 {
		t.Fatalf("expected the down alert, got %d sends", got)
	}

	if err := m.CheckIn("web1", "tok1"); err != nil {
		t.Fatalf("CheckIn: %v", err)
	}
	if got := sender.count(); got != 2 {
		t.Fatalf("expected a recovery email, got %d sends", got)
	}

	st, err := state.Load(statePath)
	if err != nil {
		t.Fatalf("state.Load: %v", err)
	}
	if st["web1"].Status != state.Up {
		t.Fatalf("expected web1 to be up after check-in, got %s", st["web1"].Status)
	}

	// A second, immediate check-in should not fire another recovery email
	// since it was already up.
	if err := m.CheckIn("web1", "tok1"); err != nil {
		t.Fatalf("CheckIn: %v", err)
	}
	if got := sender.count(); got != 2 {
		t.Fatalf("expected no extra email for a check-in while already up, got %d", got)
	}
}
