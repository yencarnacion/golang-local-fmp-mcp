package tools

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"golang-local-fmp-mcp/internal/fmp"
)

// Handler executes a tool call. It receives the raw arguments map and is
// expected to validate, build the FMP request, and return the parsed JSON.
type Handler func(ctx context.Context, client *fmp.Client, args map[string]any) (any, error)

// Tool is a single MCP tool entry.
type Tool struct {
	Name        string
	Description string
	InputSchema map[string]any
	Handler     Handler
}

// Registry holds all registered tools.
type Registry struct {
	tools []Tool
	index map[string]Tool
}

// NewRegistry builds a registry pre-populated with every FMP tool.
func NewRegistry() *Registry {
	r := &Registry{index: map[string]Tool{}}
	for _, t := range allTools() {
		r.tools = append(r.tools, t)
		r.index[t.Name] = t
	}
	return r
}

// Get fetches a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.index[name]
	return t, ok
}

// Manifest returns the tools formatted for the MCP tools/list response.
func (r *Registry) Manifest() []map[string]any {
	out := make([]map[string]any, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"inputSchema": t.InputSchema,
		})
	}
	return out
}

// --- argument helpers ------------------------------------------------------

// argString extracts a non-empty string argument by key.
func argString(args map[string]any, key string) (string, bool) {
	v, ok := args[key]
	if !ok || v == nil {
		return "", false
	}
	switch s := v.(type) {
	case string:
		s = strings.TrimSpace(s)
		return s, s != ""
	case float64:
		return strconv.FormatFloat(s, 'f', -1, 64), true
	case int:
		return strconv.Itoa(s), true
	case bool:
		return strconv.FormatBool(s), true
	default:
		return fmt.Sprintf("%v", s), true
	}
}

// argNumber extracts a numeric argument as string for use in query params.
func argNumber(args map[string]any, key string) (string, bool) {
	v, ok := args[key]
	if !ok || v == nil {
		return "", false
	}
	switch n := v.(type) {
	case float64:
		if n == float64(int64(n)) {
			return strconv.FormatInt(int64(n), 10), true
		}
		return strconv.FormatFloat(n, 'f', -1, 64), true
	case int:
		return strconv.Itoa(n), true
	case int64:
		return strconv.FormatInt(n, 10), true
	case string:
		s := strings.TrimSpace(n)
		return s, s != ""
	default:
		return fmt.Sprintf("%v", n), true
	}
}

// argBool extracts a boolean argument as a query-string value ("true"/"false").
func argBool(args map[string]any, key string) (string, bool) {
	v, ok := args[key]
	if !ok || v == nil {
		return "", false
	}
	switch b := v.(type) {
	case bool:
		return strconv.FormatBool(b), true
	case string:
		s := strings.TrimSpace(b)
		return s, s != ""
	default:
		return fmt.Sprintf("%v", b), true
	}
}

// argStringSlice extracts a list of strings.
func argStringSlice(args map[string]any, key string) ([]string, bool) {
	v, ok := args[key]
	if !ok || v == nil {
		return nil, false
	}
	switch s := v.(type) {
	case []any:
		out := make([]string, 0, len(s))
		for _, x := range s {
			if str, ok := x.(string); ok && str != "" {
				out = append(out, str)
			}
		}
		return out, len(out) > 0
	case []string:
		return s, len(s) > 0
	case string:
		// allow comma-separated string fallback
		parts := strings.Split(s, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out, len(out) > 0
	default:
		return nil, false
	}
}

// requireEndpoint returns the "endpoint" arg or an error.
func requireEndpoint(args map[string]any, allowed []string) (string, error) {
	ep, ok := argString(args, "endpoint")
	if !ok {
		return "", fmt.Errorf("missing required argument: endpoint")
	}
	for _, a := range allowed {
		if a == ep {
			return ep, nil
		}
	}
	return "", fmt.Errorf("unknown endpoint %q; expected one of: %s", ep, strings.Join(allowed, ", "))
}

// forwardArgs copies select string/number/bool args from args into params.
// `from_date` and `to_date` are translated to FMP's `from`/`to` query names.
// Slice args declared as `symbols` are joined with commas.
func forwardArgs(args map[string]any, keys []string) url.Values {
	params := url.Values{}
	for _, key := range keys {
		switch key {
		case "from_date":
			if v, ok := argString(args, "from_date"); ok {
				params.Set("from", v)
			}
		case "to_date":
			if v, ok := argString(args, "to_date"); ok {
				params.Set("to", v)
			}
		case "symbols":
			if list, ok := argStringSlice(args, "symbols"); ok {
				params.Set("symbols", strings.Join(list, ","))
			}
		default:
			if v, ok := argString(args, key); ok {
				params.Set(key, v)
			}
		}
	}
	return params
}

// commonSchema builds an MCP input schema with a required `endpoint` enum and
// the listed extra properties.
func commonSchema(endpoints []string, props map[string]any) map[string]any {
	properties := map[string]any{
		"endpoint": map[string]any{
			"type":        "string",
			"description": "The specific endpoint to call.",
			"enum":        endpoints,
		},
	}
	for k, v := range props {
		properties[k] = v
	}
	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             []string{"endpoint"},
		"additionalProperties": false,
	}
}

// Common property shapes.
var (
	propString = map[string]any{"type": "string"}
	propNumber = map[string]any{"type": "number"}
	propBool   = map[string]any{"type": "boolean"}
	propStrArr = map[string]any{"type": "array", "items": map[string]any{"type": "string"}}
)
