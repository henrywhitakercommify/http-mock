package http

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"mime"
	"net/http"
	"strings"
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

func (h *HTTP) buildHandler(endpoint config.Endpoint) (http.HandlerFunc, error) {
	slog := h.logger

	tmpl, err := template.New(endpoint.Path).Parse(endpoint.Response.Body)
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

		for key, val := range endpoint.Response.Headers {
			w.Header().Add(key, val)
		}
		code := http.StatusOK
		if endpoint.Response.Code != 0 {
			code = endpoint.Response.Code
		}
		w.WriteHeader(code)
		_, _ = w.Write(buf.Bytes())

		dur := time.Since(start)
		slog.Debug(
			"processed request",
			"method",
			r.Method,
			"path",
			r.URL.Path,
			"duration",
			dur.String(),
			"duration_ms",
			dur.Milliseconds(),
		)
		h.requestsSeconds.WithLabelValues(
			r.URL.Path,
			r.Method,
		).Observe(dur.Seconds())
		h.requestsCount.WithLabelValues(
			r.URL.Path,
			r.Method,
			fmt.Sprintf("%d", code),
		).Inc()
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
			rd.Body = parseBody(r.Header.Get("Content-Type"), body)
		}
	}

	return rd
}

func parseBody(contentType string, body []byte) map[string]any {
	mediaType, _, _ := mime.ParseMediaType(contentType)

	switch mediaType {
	case "application/xml", "text/xml":
		parsed, err := xmlToMap(body)
		if err == nil {
			return parsed
		}
	default:
		var parsed map[string]any
		if json.Unmarshal(body, &parsed) == nil {
			return parsed
		}
	}

	return nil
}

func xmlToMap(data []byte) (map[string]any, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	result, err := xmlDecodeElement(decoder, "")
	if err != nil {
		return nil, err
	}
	// Unwrap the root element if there's exactly one top-level key
	if len(result) == 1 {
		for _, v := range result {
			if inner, ok := v.(map[string]any); ok {
				return inner, nil
			}
		}
	}
	return result, nil
}

func xmlDecodeElement(decoder *xml.Decoder, parent string) (map[string]any, error) {
	result := map[string]any{}

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			child, err := xmlDecodeElement(decoder, t.Name.Local)
			if err != nil {
				return nil, err
			}
			// If the child map has only a single "" key, it's a text-only element
			if text, ok := child[""]; ok && len(child) == 1 {
				result[t.Name.Local] = text
			} else {
				result[t.Name.Local] = child
			}
		case xml.CharData:
			text := strings.TrimSpace(string(t))
			if text != "" {
				result[""] = text
			}
		case xml.EndElement:
			return result, nil
		}
	}

	return result, nil
}
