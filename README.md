# http-mock

A configurable HTTP mock server. Define endpoints in a YAML file and get a server that responds with templated content, custom headers, status codes, and simulated latency.

## Usage

```sh
go run . --config http-mock.yaml
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-c`, `--config` | `http-mock.yaml` | Path to the config file |
| `--log-level` | `info` | Log level (`debug`, `info`, `warn`, `error`) |

## Configuration

```yaml
---
endpoints:
  - path: /api/users
    status_code: 200
    response:
      headers:
        Content-Type: application/json
      body: '{"method": "{{.Request.Method}}", "name": "{{.Request.Body.name}}"}'
    minDelay: 10ms
    maxDelay: 2s
    p95Delay: 200ms
```

### Endpoint fields

| Field | Required | Description |
|-------|----------|-------------|
| `path` | yes | The URL path to match |
| `status_code` | yes | HTTP status code to return |
| `response.body` | yes | Response body (supports Go templates) |
| `response.headers` | no | Map of response headers to set |
| `minDelay` | no | Minimum delay before responding |
| `maxDelay` | no | Maximum delay before responding (requires `p95Delay`) |
| `p95Delay` | no | 95th percentile delay target (requires `maxDelay`) |

### Response templating

Response bodies are rendered using Go's `text/template` package. The following data is available under `.Request`:

| Field | Type | Description |
|-------|------|-------------|
| `.Request.Method` | `string` | HTTP method (GET, POST, etc.) |
| `.Request.Path` | `string` | Request URL path |
| `.Request.Host` | `string` | Request host |
| `.Request.Headers` | `http.Header` | Request headers (multi-valued, use `index .Request.Headers.Name 0`) |
| `.Request.Query` | `map[string][]string` | Query parameters (multi-valued, use `index .Request.Query.key 0`) |
| `.Request.Body` | `map[string]any` | Parsed request body |

Body parsing is based on the `Content-Type` header:

- `application/json` (or no Content-Type) - parsed as JSON
- `application/xml`, `text/xml` - parsed as XML (root element is unwrapped)

### Simulated latency

When both `maxDelay` and `p95Delay` are set, each response is delayed by a random duration. The distribution is skewed so that 95% of delays fall at or below the `p95Delay` value, with `maxDelay` as the absolute ceiling. `minDelay` sets a floor if provided.

## Testing

```sh
go test ./...
```
