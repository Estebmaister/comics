package health

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/rs/zerolog/log"
)

const (
	// healthCheckInterval is the interval at which the health check is performed
	healthCheckInterval = 5 * time.Second
	// readinessMaxInterval is the interval at which the readiness check is performed
	readinessMaxInterval = 1 * time.Second
)

var (
	statusUP   = []byte(`{"status":"UP"}`)
	statusDOWN = []byte(`{"status":"DOWN"}`)
)

// Pinger defines the interface for pinging a database
type Pinger interface {
	Ping(ctx context.Context) error
}

// HealthChecker checks the health of the application
type HealthChecker struct {
	db              Pinger
	isReady         atomic.Bool
	shutdownChan    chan struct{}
	manualCheckChan chan struct{}
}

func NewHealthChecker(db Pinger) *HealthChecker {
	h := &HealthChecker{
		db:              db,
		shutdownChan:    make(chan struct{}),
		manualCheckChan: make(chan struct{}),
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

// LivenessHandler returns an HTTP handler for liveness probe
func (h *HealthChecker) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(statusUP)
	}
}

// ReadinessHandler returns an HTTP handler for readiness probe
func (h *HealthChecker) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if !h.isReady.Load() {
			h.triggerManualCheck()
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write(statusDOWN)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(statusUP)
	}
}

// readinessLoop periodically checks if the service is ready
func (h *HealthChecker) readinessLoop() {
	// Create an exponential backoff configuration
	expBackoff := backoff.NewExponentialBackOff(
		backoff.WithMaxElapsedTime(0),
		backoff.WithInitialInterval(healthCheckInterval),
		backoff.WithMultiplier(1.5),
		backoff.WithMaxInterval(readinessMaxInterval),
	)

	for {
		select {
		case <-h.shutdownChan:
			return
		case <-time.After(expBackoff.NextBackOff()):
			h.performHealthCheck()
			if h.isReady.Load() {
				expBackoff.Reset()
			}
		case <-h.manualCheckChan:
			h.performHealthCheck()
			if h.isReady.Load() {
				expBackoff.Reset()
			}

		}
	}
}

// performHealthCheck checks the health of the database and store if its ready
func (h *HealthChecker) performHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), healthCheckInterval)
	defer cancel()

	err := h.db.Ping(ctx)
	h.isReady.Store(err == nil)
	if err != nil {
		log.Error().Err(err).Msg("Database health check failed")
	}
}

// triggerManualCheck signals the health checker to perform a manual check
func (h *HealthChecker) triggerManualCheck() {
	select {
	case h.manualCheckChan <- struct{}{}:
		log.Debug().Msg("Triggered manual health check")
	default:
		// If the channel is full, don't block
	}
}
