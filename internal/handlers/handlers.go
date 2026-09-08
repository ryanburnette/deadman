// Package handlers wires the check-in HTTP endpoint to the monitor.
package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ryanburnette/deadman/internal/monitor"
)

// Register mounts the check-in and health routes on r.
func Register(r chi.Router, m *monitor.Monitor) {
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	r.Get("/checkin/{name}/{token}", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		token := chi.URLParam(r, "token")
		if err := m.CheckIn(name, token); err != nil {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("not found"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
}
