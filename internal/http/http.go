// Package http
package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/henrywhitakercommify/http-mock/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type HTTP struct {
	server *http.Server
	logger *slog.Logger

	requestsSeconds *prometheus.HistogramVec
	requestsCount   *prometheus.CounterVec
}

func New(endpoints []config.Endpoint) (*HTTP, error) {
	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:    ":12345",
		Handler: mux,
	}

	slog := slog.With("component", "http")

	out := &HTTP{
		server: srv,
		logger: slog,
	}

	for _, e := range endpoints {
		handler, err := out.buildHandler(e)
		if err != nil {
			return nil, fmt.Errorf("build handler: %w", err)
		}
		mux.Handle(e.Path, handler)
	}

	mux.HandleFunc("/healthz", healthy())
	mux.HandleFunc("/readyz", ready())
	mux.Handle("/metrics", promhttp.Handler())

	reqeustsSeconds := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "The number of seconds a request took to process",
		},
		[]string{"path", "method"},
	)
	requestsCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "The total number of requests",
		},
		[]string{"path", "method", "code"},
	)

	prometheus.Register(reqeustsSeconds)
	prometheus.Register(requestsCount)
	out.requestsSeconds = reqeustsSeconds
	out.requestsCount = requestsCount

	return out, nil
}

func (h *HTTP) start() error {
	if err := h.server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("start http server: %w", err)
		}
	}
	return nil
}

func (h *HTTP) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	return h.server.Shutdown(ctx)
}

func healthy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}

func ready() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}

func (h *HTTP) Run(ctx context.Context) error {
	h.logger.Info("starting http server", "addr", h.server.Addr)
	errCh := make(chan error, 1)
	go func() {
		if err := h.start(); err != nil {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		h.logger.Error("failed to start http server", "error", err)
		return err
	case <-ctx.Done():
		break
	}

	h.logger.Info("shutting down http server")
	return h.shutdown()
}
