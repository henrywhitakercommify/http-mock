package http

import (
	"log/slog"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/henrywhitakercommify/http-mock/internal/config"
)

func buildHandler(endpoint config.Endpoint, slog *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		if endpoint.MinDelay != 0 && endpoint.MaxDelay != 0 {
			<-time.After(randRange(endpoint.MinDelay, endpoint.MaxDelay))
		}
		w.WriteHeader(endpoint.StatusCode)
		_, _ = w.Write([]byte(endpoint.Response))

		dur := time.Since(start)
		slog.Debug(
			"processed request",
			"path",
			endpoint.Path,
			"duration",
			dur.String(),
			"duration_ms",
			dur.Milliseconds(),
		)
	}
}

func randRange(min, max time.Duration) time.Duration {
	return time.Duration(rand.IntN(int(max)-int(min)) + int(min))
}
