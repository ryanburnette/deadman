package mailer

import (
	"strings"
	"testing"
	"time"
)

func TestDownEmailOnce(t *testing.T) {
	lastSeen := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	subject, body := DownEmail("web1", lastSeen, 5*time.Minute, time.Minute, 0, 6*time.Minute)

	if !strings.Contains(subject, "web1") {
		t.Errorf("subject %q should mention service name", subject)
	}
	if strings.Contains(body, "keep alerting") {
		t.Errorf("body should not mention repeat alerts when repeat is 0: %q", body)
	}
	if !strings.Contains(body, "6m0s") {
		t.Errorf("body should mention how long it's been silent: %q", body)
	}
}

func TestDownEmailRepeat(t *testing.T) {
	lastSeen := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	_, body := DownEmail("web1", lastSeen, 5*time.Minute, time.Minute, time.Hour, 6*time.Minute)

	if !strings.Contains(body, "keep alerting every 1h0m0s") {
		t.Errorf("body should mention the repeat interval: %q", body)
	}
}

func TestUpEmail(t *testing.T) {
	subject, body := UpEmail("web1", 90*time.Second)

	if !strings.Contains(subject, "web1") || !strings.Contains(subject, "back up") {
		t.Errorf("unexpected subject: %q", subject)
	}
	if !strings.Contains(body, "1m30s") {
		t.Errorf("body should mention downtime: %q", body)
	}
}
