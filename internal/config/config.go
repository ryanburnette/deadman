// Package config manages the services.csv file: the list of monitored
// services and their heartbeat expectations. It is only ever mutated
// through the deadman CLI, never hand-edited, so the on-disk format can stay
// a minimal CSV.
package config

import (
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

var header = []string{"name", "token", "interval", "grace", "repeat", "email"}

// Service is one monitored heartbeat.
type Service struct {
	Name     string
	Token    string
	Interval time.Duration
	Grace    time.Duration
	// Repeat is how often to resend the down alert while a service stays
	// down. Zero means send a single alert and stay silent until recovery.
	Repeat time.Duration
	Email  string
}

// Deadline is how long after a checkin the service may go silent before
// it's considered down.
func (s Service) Deadline() time.Duration {
	return s.Interval + s.Grace
}

// Load reads services.csv. A missing file is treated as an empty config.
func Load(path string) ([]Service, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	if len(rows) == 0 {
		return nil, nil
	}

	services := make([]Service, 0, len(rows)-1)
	for _, row := range rows[1:] {
		if len(row) != len(header) {
			return nil, fmt.Errorf("%s: malformed row %q", path, row)
		}
		interval, err := time.ParseDuration(row[2])
		if err != nil {
			return nil, fmt.Errorf("%s: service %q: bad interval %q: %w", path, row[0], row[2], err)
		}
		grace, err := time.ParseDuration(row[3])
		if err != nil {
			return nil, fmt.Errorf("%s: service %q: bad grace %q: %w", path, row[0], row[3], err)
		}
		repeat, err := time.ParseDuration(row[4])
		if err != nil {
			return nil, fmt.Errorf("%s: service %q: bad repeat %q: %w", path, row[0], row[4], err)
		}
		services = append(services, Service{
			Name:     row[0],
			Token:    row[1],
			Interval: interval,
			Grace:    grace,
			Repeat:   repeat,
			Email:    row[5],
		})
	}
	return services, nil
}

// Save writes services.csv atomically (write to a temp file, then rename),
// sorted by name for a stable diff.
func Save(path string, services []Service) error {
	sorted := make([]Service, len(services))
	copy(sorted, services)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
	}

	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("creating %s: %w", tmp, err)
	}

	w := csv.NewWriter(f)
	if err := w.Write(header); err != nil {
		f.Close()
		return fmt.Errorf("writing %s: %w", tmp, err)
	}
	for _, s := range sorted {
		row := []string{
			s.Name,
			s.Token,
			s.Interval.String(),
			s.Grace.String(),
			s.Repeat.String(),
			s.Email,
		}
		if err := w.Write(row); err != nil {
			f.Close()
			return fmt.Errorf("writing %s: %w", tmp, err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		f.Close()
		return fmt.Errorf("writing %s: %w", tmp, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("renaming %s to %s: %w", tmp, path, err)
	}
	return nil
}

// NewToken generates a random per-service check-in token.
func NewToken() (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", fmt.Errorf("generating token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Find returns the service with the given name, if any.
func Find(services []Service, name string) (Service, bool) {
	for _, s := range services {
		if s.Name == name {
			return s, true
		}
	}
	return Service{}, false
}
