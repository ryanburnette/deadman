package state

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "state.json")

	want := State{
		"web1": {LastSeen: time.Now().Truncate(time.Second), Status: Up},
		"web2": {LastSeen: time.Now().Truncate(time.Second), Status: Down, LastNotified: time.Now().Truncate(time.Second)},
	}

	if err := Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d", len(got), len(want))
	}
	for name, w := range want {
		g, ok := got[name]
		if !ok {
			t.Errorf("missing entry for %s", name)
			continue
		}
		if !g.LastSeen.Equal(w.LastSeen) || g.Status != w.Status || !g.LastNotified.Equal(w.LastNotified) {
			t.Errorf("%s: got %+v, want %+v", name, g, w)
		}
	}
}

func TestLoadMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.json")

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %v, want empty state for missing file", got)
	}
}
