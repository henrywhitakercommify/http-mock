package http

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/henrywhitakercommify/http-mock/internal/config"
)

func TestRandDelay(t *testing.T) {
	t.Run("returns values within bounds", func(t *testing.T) {
		min := 100 * time.Millisecond
		max := 5 * time.Second
		p95 := 500 * time.Millisecond

		for range 1000 {
			d := randDelay(min, max, p95)
			if d < min {
				t.Fatalf("got %v, want >= %v", d, min)
			}
			if d > min+max {
				t.Fatalf("got %v, want <= %v", d, min+max)
			}
		}
	})

	t.Run("95 percent of values fall at or below p95", func(t *testing.T) {
		min := time.Duration(0)
		max := 5 * time.Second
		p95 := 500 * time.Millisecond

		n := 10000
		below := 0
		for range n {
			if randDelay(min, max, p95) <= p95 {
				below++
			}
		}

		ratio := float64(below) / float64(n)
		// Allow some statistical tolerance
		if ratio < 0.93 || ratio > 0.97 {
			t.Fatalf("expected ~95%% of values <= p95, got %.1f%%", ratio*100)
		}
	})
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestBuildHandler_StaticResponse(t *testing.T) {
	endpoint := config.Endpoint{
		Path:       "/test",
		Response:   "hello world",
		StatusCode: 200,
	}

	handler, err := buildHandler(endpoint, testLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != 200 {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
	if rec.Body.String() != "hello world" {
		t.Fatalf("got body %q, want %q", rec.Body.String(), "hello world")
	}
}

func TestBuildHandler_TemplateMethod(t *testing.T) {
	endpoint := config.Endpoint{
		Path:       "/method",
		Response:   "method: {{.Request.Method}}",
		StatusCode: 200,
	}

	handler, err := buildHandler(endpoint, testLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/method", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Body.String() != "method: POST" {
		t.Fatalf("got body %q, want %q", rec.Body.String(), "method: POST")
	}
}

func TestBuildHandler_TemplateQuery(t *testing.T) {
	endpoint := config.Endpoint{
		Path:       "/query",
		Response:   "name: {{index .Request.Query.name 0}}",
		StatusCode: 200,
	}

	handler, err := buildHandler(endpoint, testLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/query?name=alice", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Body.String() != "name: alice" {
		t.Fatalf("got body %q, want %q", rec.Body.String(), "name: alice")
	}
}

func TestBuildHandler_TemplateJSONBody(t *testing.T) {
	endpoint := config.Endpoint{
		Path:       "/body",
		Response:   "greeting: {{.Request.Body.greeting}}",
		StatusCode: 200,
	}

	handler, err := buildHandler(endpoint, testLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := strings.NewReader(`{"greeting":"hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/body", body)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Body.String() != "greeting: hello" {
		t.Fatalf("got body %q, want %q", rec.Body.String(), "greeting: hello")
	}
}

func TestBuildHandler_TemplateHeaders(t *testing.T) {
	endpoint := config.Endpoint{
		Path:       "/headers",
		Response:   "auth: {{index .Request.Headers.Authorization 0}}",
		StatusCode: 200,
	}

	handler, err := buildHandler(endpoint, testLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/headers", nil)
	req.Header.Set("Authorization", "Bearer token123")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Body.String() != "auth: Bearer token123" {
		t.Fatalf("got body %q, want %q", rec.Body.String(), "auth: Bearer token123")
	}
}

func TestBuildHandler_InvalidTemplate(t *testing.T) {
	endpoint := config.Endpoint{
		Path:       "/bad",
		Response:   "{{.Invalid",
		StatusCode: 200,
	}

	_, err := buildHandler(endpoint, testLogger())
	if err == nil {
		t.Fatal("expected error for invalid template, got nil")
	}
}

func TestBuildHandler_StatusCode(t *testing.T) {
	endpoint := config.Endpoint{
		Path:       "/not-found",
		Response:   "not found",
		StatusCode: 404,
	}

	handler, err := buildHandler(endpoint, testLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/not-found", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != 404 {
		t.Fatalf("got status %d, want 404", rec.Code)
	}
}
