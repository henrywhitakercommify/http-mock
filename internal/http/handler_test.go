package http

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/henrywhitakercommify/http-mock/internal/config"
	"github.com/prometheus/client_golang/prometheus"
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

func testHTTP() *HTTP {
	return &HTTP{
		logger: testLogger(),
		requestsSeconds: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "test_requests_seconds",
			},
			[]string{"path", "method"},
		),
		requestsCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "test_requests_count",
			},
			[]string{"path", "method", "code"},
		),
	}
}

func TestBuildHandlerStaticResponse(t *testing.T) {
	endpoint := config.Endpoint{
		Path:     "/test",
		Response: config.Response{Body: "hello world", Code: 200},
	}

	handler, err := testHTTP().buildHandler(endpoint)
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

func TestBuildHandlerTemplateMethod(t *testing.T) {
	endpoint := config.Endpoint{
		Path:     "/method",
		Response: config.Response{Body: "method: {{.Request.Method}}", Code: 200},
	}

	handler, err := testHTTP().buildHandler(endpoint)
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

func TestBuildHandlerTemplateQuery(t *testing.T) {
	endpoint := config.Endpoint{
		Path:     "/query",
		Response: config.Response{Body: "name: {{index .Request.Query.name 0}}", Code: 200},
	}

	handler, err := testHTTP().buildHandler(endpoint)
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

func TestBuildHandlerTemplateJSONBody(t *testing.T) {
	endpoint := config.Endpoint{
		Path:     "/body",
		Response: config.Response{Body: "greeting: {{.Request.Body.greeting}}", Code: 200},
	}

	handler, err := testHTTP().buildHandler(endpoint)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := strings.NewReader(`{"greeting":"hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/body", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Body.String() != "greeting: hello" {
		t.Fatalf("got body %q, want %q", rec.Body.String(), "greeting: hello")
	}
}

func TestBuildHandlerTemplateXMLBody(t *testing.T) {
	endpoint := config.Endpoint{
		Path:     "/xml",
		Response: config.Response{Body: "greeting: {{.Request.Body.greeting}}", Code: 200},
	}

	handler, err := testHTTP().buildHandler(endpoint)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := strings.NewReader(`<root><greeting>hello</greeting></root>`)
	req := httptest.NewRequest(http.MethodPost, "/xml", body)
	req.Header.Set("Content-Type", "application/xml")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Body.String() != "greeting: hello" {
		t.Fatalf("got body %q, want %q", rec.Body.String(), "greeting: hello")
	}
}

func TestBuildHandlerTemplateXMLNestedBody(t *testing.T) {
	endpoint := config.Endpoint{
		Path:     "/xml-nested",
		Response: config.Response{Body: "city: {{.Request.Body.address.city}}", Code: 200},
	}

	handler, err := testHTTP().buildHandler(endpoint)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := strings.NewReader(`<root><address><city>London</city></address></root>`)
	req := httptest.NewRequest(http.MethodPost, "/xml-nested", body)
	req.Header.Set("Content-Type", "text/xml")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Body.String() != "city: London" {
		t.Fatalf("got body %q, want %q", rec.Body.String(), "city: London")
	}
}

func TestBuildHandlerTemplateXMLWithCharset(t *testing.T) {
	endpoint := config.Endpoint{
		Path:     "/xml-charset",
		Response: config.Response{Body: "name: {{.Request.Body.name}}", Code: 200},
	}

	handler, err := testHTTP().buildHandler(endpoint)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := strings.NewReader(`<root><name>alice</name></root>`)
	req := httptest.NewRequest(http.MethodPost, "/xml-charset", body)
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Body.String() != "name: alice" {
		t.Fatalf("got body %q, want %q", rec.Body.String(), "name: alice")
	}
}

func TestBuildHandlerJSONFallbackWithNoContentType(t *testing.T) {
	endpoint := config.Endpoint{
		Path:     "/no-ct",
		Response: config.Response{Body: "val: {{.Request.Body.key}}", Code: 200},
	}

	handler, err := testHTTP().buildHandler(endpoint)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := strings.NewReader(`{"key":"value"}`)
	req := httptest.NewRequest(http.MethodPost, "/no-ct", body)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Body.String() != "val: value" {
		t.Fatalf("got body %q, want %q", rec.Body.String(), "val: value")
	}
}

func TestBuildHandlerTemplateHeaders(t *testing.T) {
	endpoint := config.Endpoint{
		Path:     "/headers",
		Response: config.Response{Body: "auth: {{index .Request.Headers.Authorization 0}}", Code: 200},
	}

	handler, err := testHTTP().buildHandler(endpoint)
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

func TestBuildHandlerTemplateRandstr(t *testing.T) {
	endpoint := config.Endpoint{
		Path:     "/rand",
		Response: config.Response{Body: `{{randstr 16}}`, Code: 200},
	}

	handler, err := testHTTP().buildHandler(endpoint)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/rand", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	got := rec.Body.String()
	if len(got) != 16 {
		t.Fatalf("got length %d, want 16", len(got))
	}
	if match, _ := regexp.MatchString(`^[0-9a-f]{16}$`, got); !match {
		t.Fatalf("got %q, want hex string", got)
	}

	// Verify different calls produce different values
	rec2 := httptest.NewRecorder()
	handler(rec2, httptest.NewRequest(http.MethodGet, "/rand", nil))
	if rec.Body.String() == rec2.Body.String() {
		t.Fatal("expected different random strings across calls")
	}
}

func TestBuildHandlerTemplateUUID(t *testing.T) {
	endpoint := config.Endpoint{
		Path:     "/uuid",
		Response: config.Response{Body: `{{uuid}}`, Code: 200},
	}

	handler, err := testHTTP().buildHandler(endpoint)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/uuid", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	got := rec.Body.String()
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	if !uuidRegex.MatchString(got) {
		t.Fatalf("got %q, want valid UUID format", got)
	}

	// Verify different calls produce different UUIDs
	rec2 := httptest.NewRecorder()
	handler(rec2, httptest.NewRequest(http.MethodGet, "/uuid", nil))
	if rec.Body.String() == rec2.Body.String() {
		t.Fatal("expected different UUIDs across calls")
	}
}

func TestBuildHandlerTemplateRandstrAndUUIDInBody(t *testing.T) {
	endpoint := config.Endpoint{
		Path:     "/combined",
		Response: config.Response{Body: `{"id":"{{uuid}}","token":"{{randstr 32}}"}`, Code: 200},
	}

	handler, err := testHTTP().buildHandler(endpoint)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/combined", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	got := rec.Body.String()
	combined := regexp.MustCompile(`^\{"id":"[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}","token":"[0-9a-f]{32}"\}$`)
	if !combined.MatchString(got) {
		t.Fatalf("got %q, want JSON with uuid and randstr", got)
	}
}

func TestBuildHandlerInvalidTemplate(t *testing.T) {
	endpoint := config.Endpoint{
		Path:     "/bad",
		Response: config.Response{Body: "{{.Invalid", Code: 200},
	}

	_, err := testHTTP().buildHandler(endpoint)
	if err == nil {
		t.Fatal("expected error for invalid template, got nil")
	}
}

func TestBuildHandlerStatusCode(t *testing.T) {
	endpoint := config.Endpoint{
		Path:     "/not-found",
		Response: config.Response{Body: "not found", Code: 404},
	}

	handler, err := testHTTP().buildHandler(endpoint)
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
