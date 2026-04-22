package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"text/template"
	"time"

	"github.com/henrywhitakercommify/http-mock/internal/config"
)

// requestData is the template context exposed as .Request
type requestData struct {
	Method  string
	Path    string
	Host    string
	Headers http.Header
	Query   map[string][]string
	Body    map[string]any
}

func newRequestData(r *http.Request) requestData {
	rd := requestData{
		Method:  r.Method,
		Path:    r.URL.Path,
		Host:    r.Host,
		Headers: r.Header,
		Query:   r.URL.Query(),
	}

	if r.Body != nil {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err == nil && len(body) > 0 {
			var parsed map[string]any
			if json.Unmarshal(body, &parsed) == nil {
				rd.Body = parsed
			}
		}
	}

	return rd
}

func buildHandler(endpoint config.Endpoint, slog *slog.Logger) (http.HandlerFunc, error) {
	tmpl, err := template.New(endpoint.Path).Parse(endpoint.Response)
	if err != nil {
		return nil, fmt.Errorf("build response template: %w", err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		if endpoint.MaxDelay != 0 && endpoint.P95Delay != 0 {
			<-time.After(randDelay(endpoint.MinDelay, endpoint.MaxDelay, endpoint.P95Delay))
		}

		data := struct {
			Request requestData
		}{
			Request: newRequestData(r),
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			slog.Error("failed to execute response template", "path", endpoint.Path, "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(endpoint.StatusCode)
		_, _ = w.Write(buf.Bytes())

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
	}, nil
}

// randDelay returns a random duration in [0, maxDelay] where 95% of
// values fall at or below p95. It uses a power distribution with the
// exponent derived from: k = ln(p95/max) / ln(0.95).
func randDelay(minDelay, maxDelay, p95 time.Duration) time.Duration {
	k := math.Log(float64(p95)/float64(maxDelay)) / math.Log(0.95)
	r := rand.Float64()
	return time.Duration(float64(minDelay) + (math.Pow(r, k) * float64(maxDelay)))
}
