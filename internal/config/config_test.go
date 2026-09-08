package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "services.csv")

	want := []Service{
		{Name: "web1", Token: "tok1", Interval: 5 * time.Minute, Grace: time.Minute, Repeat: 0, Email: "a@example.com"},
		{Name: "web2", Token: "tok2", Interval: 30 * time.Second, Grace: 10 * time.Second, Repeat: time.Hour, Email: "b@example.com"},
	}

	if err := Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("got %d services, want %d", len(got), len(want))
	}
	// Save sorts by name, so web1 should come before web2.
	for i, w := range want {
		g := got[i]
		if g != w {
			t.Errorf("service %d: got %+v, want %+v", i, g, w)
		}
	}
}

func TestLoadMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.csv")

	services, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if services != nil {
		t.Fatalf("got %v, want nil for missing file", services)
	}
}

func TestLoadMalformedDuration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "services.csv")
	content := "name,token,interval,grace,repeat,email\nweb1,tok1,not-a-duration,1m0s,0s,a@example.com\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("expected an error for a malformed interval, got nil")
	}
}

func TestFind(t *testing.T) {
	services := []Service{
		{Name: "web1"},
		{Name: "web2"},
	}

	if _, ok := Find(services, "web1"); !ok {
		t.Error("expected to find web1")
	}
	if _, ok := Find(services, "missing"); ok {
		t.Error("expected not to find a service that doesn't exist")
	}
}
