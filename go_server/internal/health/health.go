package health

import (
	"context"
	"database/sql"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

type HealthChecker struct {
	db           *sql.DB
	isReady      atomic.Bool
	shutdownChan chan struct{}
}

func NewHealthChecker(db *sql.DB) *HealthChecker {
	h := &HealthChecker{
		db:           db,
		shutdownChan: make(chan struct{}),
	}
	h.isReady.Store(false)
	return h
}

// Start begins the readiness check loop
func (h *HealthChecker) Start() {
	go h.readinessLoop()
}

// Stop signals the health checker to stop
func (h *HealthChecker) Stop() {
	close(h.shutdownChan)
}

// readinessLoop periodically checks if the service is ready
func (h *HealthChecker) readinessLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.shutdownChan:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			err := h.db.PingContext(ctx)
			cancel()

			h.isReady.Store(err == nil)
			if err != nil {
				log.Error().Err(err).Msg("Database health check failed")
			}
		}
	}
}

// LivenessHandler returns an HTTP handler for liveness probe
func (h *HealthChecker) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"UP"}`))
	}
}

// ReadinessHandler returns an HTTP handler for readiness probe
func (h *HealthChecker) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		if !h.isReady.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"DOWN"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"UP"}`))
	}
}
