package tunnel

import (
	"context"
	"sync"
	"time"

	"github.com/labbs/sshtunnel/internal/config"
)

// ReconnectConfig contains reconnection settings
type ReconnectConfig struct {
	Enabled           bool
	MaxAttempts       int
	InitialDelay      time.Duration
	MaxDelay          time.Duration
	BackoffMultiplier float64
}

// NewReconnectConfig creates a ReconnectConfig from config
func NewReconnectConfig(cfg config.ReconnectConfig) ReconnectConfig {
	return ReconnectConfig{
		Enabled:           cfg.Enabled,
		MaxAttempts:       cfg.MaxAttempts,
		InitialDelay:      cfg.InitialDelay,
		MaxDelay:          cfg.MaxDelay,
		BackoffMultiplier: cfg.BackoffMultiplier,
	}
}

// Reconnector handles automatic reconnection for tunnels
type Reconnector struct {
	mu      sync.Mutex
	config  ReconnectConfig
	tunnel  Tunnel
	attempt int
}

// NewReconnector creates a new reconnector
func NewReconnector(cfg ReconnectConfig, tunnel Tunnel) *Reconnector {
	return &Reconnector{
		config: cfg,
		tunnel: tunnel,
	}
}

// Start starts the reconnection loop
func (r *Reconnector) Start(ctx context.Context) {
	if !r.config.Enabled {
		return
	}

	go r.reconnectLoop(ctx)
}

// reconnectLoop handles the reconnection logic
func (r *Reconnector) reconnectLoop(ctx context.Context) {
	delay := r.config.InitialDelay

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		status := r.tunnel.Status()

		// Only attempt reconnect if tunnel is in error state or disconnected unexpectedly
		if status != StatusError && status != StatusReconnecting {
			// Reset attempt counter when connected
			if status == StatusRunning {
				r.mu.Lock()
				r.attempt = 0
				r.mu.Unlock()
				delay = r.config.InitialDelay
			}
			time.Sleep(time.Second)
			continue
		}

		// Check if we've exceeded max attempts
		r.mu.Lock()
		if r.config.MaxAttempts > 0 && r.attempt >= r.config.MaxAttempts {
			r.mu.Unlock()
			return
		}
		r.attempt++
		r.mu.Unlock()
		r.tunnel.Stats().IncrementReconnectCount()

		// Attempt to restart
		err := r.tunnel.Start(ctx)
		if err == nil {
			r.mu.Lock()
			r.attempt = 0
			r.mu.Unlock()
			delay = r.config.InitialDelay
			continue
		}

		// Wait before next attempt with exponential backoff
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}

		// Increase delay for next attempt
		delay = time.Duration(float64(delay) * r.config.BackoffMultiplier)
		if delay > r.config.MaxDelay {
			delay = r.config.MaxDelay
		}
	}
}

// GetAttempt returns the current attempt number
func (r *Reconnector) GetAttempt() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.attempt
}

// GetMaxAttempts returns the max attempts configured
func (r *Reconnector) GetMaxAttempts() int {
	return r.config.MaxAttempts
}
