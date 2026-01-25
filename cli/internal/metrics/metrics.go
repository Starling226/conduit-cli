/*
 * Copyright (c) 2026, Psiphon Inc.
 * All rights reserved.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

// Package metrics provides Prometheus metrics export for the Conduit service
package metrics

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Exporter manages Prometheus metrics for the Conduit service
type Exporter struct {
	// Gauges (using Gauge instead of GaugeVec since we have no labels)
	connectingClients prometheus.Gauge
	connectedClients  prometheus.Gauge
	isLive           prometheus.Gauge

	// Counters (using Counter instead of CounterVec since we have no labels)
	totalBytesUp   prometheus.Counter
	totalBytesDown prometheus.Counter

	// Info metric for configuration (needs labels for config values)
	configInfo *prometheus.GaugeVec

	// Uptime metric
	uptimeSeconds prometheus.Gauge

	// Track previous counter values to calculate deltas
	mu            sync.Mutex
	prevBytesUp   int64
	prevBytesDown int64
	prevConfig    struct {
		maxClients    string
		bandwidthMbps string
	}

	server *http.Server
}

// Stats represents the current statistics to export
type Stats struct {
	ConnectingClients int
	ConnectedClients  int
	TotalBytesUp      int64
	TotalBytesDown    int64
	StartTime         time.Time
	IsLive            bool
	MaxClients        int
	BandwidthMbps     float64
}

// NewExporter creates a new Prometheus metrics exporter
func NewExporter() *Exporter {
	return &Exporter{
		connectingClients: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "conduit_connecting_clients",
				Help: "Number of clients currently connecting to the proxy",
			},
		),
		connectedClients: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "conduit_connected_clients",
				Help: "Number of clients currently connected to the proxy",
			},
		),
		isLive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "conduit_is_live",
				Help: "Whether the proxy is connected to the broker and ready (1) or not (0)",
			},
		),
		totalBytesUp: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "conduit_bytes_up_total",
				Help: "Total bytes uploaded (sent to clients)",
			},
		),
		totalBytesDown: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "conduit_bytes_down_total",
				Help: "Total bytes downloaded (received from clients)",
			},
		),
		configInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "conduit_config_info",
				Help: "Configuration information (max_clients, bandwidth_mbps)",
			},
			[]string{"max_clients", "bandwidth_mbps"},
		),
		uptimeSeconds: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "conduit_uptime_seconds",
				Help: "Service uptime in seconds",
			},
		),
	}
}

// Register registers all metrics with the default Prometheus registry
func (e *Exporter) Register() error {
	registry := prometheus.DefaultRegisterer
	if err := registry.Register(e.connectingClients); err != nil {
		return fmt.Errorf("failed to register connectingClients: %w", err)
	}
	if err := registry.Register(e.connectedClients); err != nil {
		return fmt.Errorf("failed to register connectedClients: %w", err)
	}
	if err := registry.Register(e.isLive); err != nil {
		return fmt.Errorf("failed to register isLive: %w", err)
	}
	if err := registry.Register(e.totalBytesUp); err != nil {
		return fmt.Errorf("failed to register totalBytesUp: %w", err)
	}
	if err := registry.Register(e.totalBytesDown); err != nil {
		return fmt.Errorf("failed to register totalBytesDown: %w", err)
	}
	if err := registry.Register(e.configInfo); err != nil {
		return fmt.Errorf("failed to register configInfo: %w", err)
	}
	if err := registry.Register(e.uptimeSeconds); err != nil {
		return fmt.Errorf("failed to register uptimeSeconds: %w", err)
	}
	return nil
}

// Update updates all metrics with the current statistics
func (e *Exporter) Update(stats Stats) {
	// Update gauges (thread-safe, no mutex needed)
	e.connectingClients.Set(float64(stats.ConnectingClients))
	e.connectedClients.Set(float64(stats.ConnectedClients))

	isLiveValue := 0.0
	if stats.IsLive {
		isLiveValue = 1.0
	}
	e.isLive.Set(isLiveValue)

	// Update counters and config info with mutex protection
	// Use a single lock to optimize mutex usage
	maxClientsStr := fmt.Sprintf("%d", stats.MaxClients)
	bandwidthStr := fmt.Sprintf("%.2f", stats.BandwidthMbps)
	
	e.mu.Lock()
	// Calculate counter deltas
	bytesUpDelta := stats.TotalBytesUp - e.prevBytesUp
	bytesDownDelta := stats.TotalBytesDown - e.prevBytesDown

	// Only add positive deltas (handle counter resets gracefully)
	// Note: Prometheus will detect counter resets (decreases) automatically
	if bytesUpDelta > 0 {
		e.totalBytesUp.Add(float64(bytesUpDelta))
		e.prevBytesUp = stats.TotalBytesUp
	} else if bytesUpDelta < 0 {
		// Counter was reset (e.g., service restart), start fresh
		// Prometheus will handle the reset by detecting the decrease
		e.prevBytesUp = stats.TotalBytesUp
	}

	if bytesDownDelta > 0 {
		e.totalBytesDown.Add(float64(bytesDownDelta))
		e.prevBytesDown = stats.TotalBytesDown
	} else if bytesDownDelta < 0 {
		// Counter was reset (e.g., service restart), start fresh
		e.prevBytesDown = stats.TotalBytesDown
	}

	// Handle config info metric - remove old label combinations if config changed
	if e.prevConfig.maxClients != "" && e.prevConfig.bandwidthMbps != "" {
		if e.prevConfig.maxClients != maxClientsStr || e.prevConfig.bandwidthMbps != bandwidthStr {
			e.configInfo.DeleteLabelValues(e.prevConfig.maxClients, e.prevConfig.bandwidthMbps)
		}
	}
	e.prevConfig.maxClients = maxClientsStr
	e.prevConfig.bandwidthMbps = bandwidthStr
	e.mu.Unlock()
	
	// Set new config info (outside mutex since WithLabelValues and Set are thread-safe)
	e.configInfo.WithLabelValues(maxClientsStr, bandwidthStr).Set(1)

	// Update uptime
	uptime := time.Since(stats.StartTime).Seconds()
	e.uptimeSeconds.Set(uptime)
}

// StartMetricsServer starts an HTTP server to expose Prometheus metrics
// address can be in the format "ip:port" or ":port" (all interfaces) or just "port" (defaults to all interfaces)
func (e *Exporter) StartMetricsServer(ctx context.Context, address string) error {
	if address == "" {
		return fmt.Errorf("metrics address cannot be empty")
	}

	// Basic validation: reject obviously invalid addresses
	if address == ":" {
		return fmt.Errorf("invalid metrics address: %q (missing port)", address)
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	// Parse address format:
	// - If it starts with ":", it's already in ":port" format
	// - If it contains ":", it's in "ip:port" format
	// - Otherwise, treat it as just a port number and prepend ":"
	addr := address
	if len(addr) > 0 && addr[0] != ':' && !strings.Contains(addr, ":") {
		// It's just a port number, prepend ":" to bind to all interfaces
		addr = ":" + addr
	}

	e.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := e.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case err := <-errChan:
		return fmt.Errorf("metrics server error: %w", err)
	case <-ctx.Done():
		// Shutdown gracefully
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := e.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("failed to shutdown metrics server: %w", err)
		}
		return nil
	}
}
