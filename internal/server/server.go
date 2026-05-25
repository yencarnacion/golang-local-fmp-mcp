package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"golang-local-fmp-mcp/internal/config"
	"golang-local-fmp-mcp/internal/fmp"
	"golang-local-fmp-mcp/internal/tools"
)

const (
	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternalError  = -32603
)

// Server is the JSON-RPC over HTTP MCP server.
type Server struct {
	cfg      *config.Config
	registry *tools.Registry
	client   *fmp.Client
	http     *http.Server
}

// New builds a server bound to cfg.
func New(cfg *config.Config) *Server {
	client := fmp.New(
		cfg.FMP.BaseURL,
		cfg.FMP.APIPath,
		cfg.APIKey,
		cfg.FMP.UserAgent,
		time.Duration(cfg.FMP.TimeoutSeconds)*time.Second,
	)
	return &Server{
		cfg:      cfg,
		registry: tools.NewRegistry(),
		client:   client,
	}
}

// Run starts listening and blocks until ctx is cancelled, at which point the
// underlying http.Server is gracefully shut down.
func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc(s.cfg.Server.MCPPath, s.handleMCP)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	s.http = &http.Server{
		Addr:              s.cfg.Addr(),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("golang-local-fmp-mcp listening on http://%s%s", s.cfg.Addr(), s.cfg.Server.MCPPath)
		if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		log.Printf("shutting down http server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.http.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		return nil
	case err, ok := <-errCh:
		if ok && err != nil {
			return err
		}
		return nil
	}
}

// --- JSON-RPC plumbing -----------------------------------------------------

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, nil, codeParseError, "read body: "+err.Error())
		return
	}
	defer r.Body.Close()

	var req jsonRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, nil, codeParseError, "parse json: "+err.Error())
		return
	}
	if req.JSONRPC != "" && req.JSONRPC != "2.0" {
		writeError(w, req.ID, codeInvalidRequest, "jsonrpc must be 2.0")
		return
	}

	start := time.Now()
	defer func() {
		log.Printf("rpc method=%s dur=%s", req.Method, time.Since(start))
	}()

	switch req.Method {
	case "initialize":
		s.handleInitialize(w, &req)
	case "tools/list":
		s.handleToolsList(w, &req)
	case "tools/call":
		s.handleToolsCall(r.Context(), w, &req)
	case "ping":
		writeResult(w, req.ID, map[string]any{"pong": true})
	default:
		writeError(w, req.ID, codeMethodNotFound, "unknown method: "+req.Method)
	}
}

func (s *Server) handleInitialize(w http.ResponseWriter, req *jsonRPCRequest) {
	writeResult(w, req.ID, map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    "golang-local-fmp-mcp",
			"version": "0.1.0",
		},
	})
}

func (s *Server) handleToolsList(w http.ResponseWriter, req *jsonRPCRequest) {
	writeResult(w, req.ID, map[string]any{"tools": s.registry.Manifest()})
}

func (s *Server) handleToolsCall(ctx context.Context, w http.ResponseWriter, req *jsonRPCRequest) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			log.Printf("tool_call_bad_params rpc_id=%s err=%q raw_params=%s", rpcID(req.ID), err.Error(), redactAndTruncate(string(req.Params), 1000))
			writeError(w, req.ID, codeInvalidParams, "parse params: "+err.Error())
			return
		}
	}
	if params.Name == "" {
		log.Printf("tool_call_bad_params rpc_id=%s err=%q raw_params=%s", rpcID(req.ID), "missing tool name", redactAndTruncate(string(req.Params), 1000))
		writeError(w, req.ID, codeInvalidParams, "missing tool name")
		return
	}
	tool, ok := s.registry.Get(params.Name)
	if !ok {
		log.Printf("tool_call_unknown_tool rpc_id=%s tool=%q args=%s", rpcID(req.ID), params.Name, safeLogArgs(params.Arguments))
		writeError(w, req.ID, codeMethodNotFound, "unknown tool: "+params.Name)
		return
	}
	if params.Arguments == nil {
		params.Arguments = map[string]any{}
	}

	callStart := time.Now()
	endpoint := argForLog(params.Arguments, "endpoint")
	out, err := tool.Handler(ctx, s.client, params.Arguments)
	if err != nil {
		log.Printf("tool_call_failed rpc_id=%s tool=%q endpoint=%q args=%s dur=%s err=%q", rpcID(req.ID), params.Name, endpoint, safeLogArgs(params.Arguments), time.Since(callStart), err.Error())
		writeResult(w, req.ID, map[string]any{
			"isError": true,
			"content": []map[string]any{{
				"type": "text",
				"text": err.Error(),
			}},
		})
		return
	}

	text, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		log.Printf("tool_call_encode_failed rpc_id=%s tool=%q endpoint=%q args=%s dur=%s err=%q", rpcID(req.ID), params.Name, endpoint, safeLogArgs(params.Arguments), time.Since(callStart), err.Error())
		writeError(w, req.ID, codeInternalError, "encode result: "+err.Error())
		return
	}
	log.Printf("tool_call_ok rpc_id=%s tool=%q endpoint=%q dur=%s", rpcID(req.ID), params.Name, endpoint, time.Since(callStart))
	writeResult(w, req.ID, map[string]any{
		"content": []map[string]any{{
			"type": "text",
			"text": string(text),
		}},
	})
}

func writeResult(w http.ResponseWriter, id json.RawMessage, result any) {
	writeJSON(w, jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: result})
}

func writeError(w http.ResponseWriter, id json.RawMessage, code int, msg string) {
	log.Printf("rpc_error id=%s code=%d msg=%q", rpcID(id), code, msg)
	writeJSON(w, jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &jsonRPCError{Code: code, Message: msg},
	})
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if err := enc.Encode(payload); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func rpcID(id json.RawMessage) string {
	if len(id) == 0 {
		return "null"
	}
	return redactAndTruncate(string(id), 200)
}

func argForLog(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	v, ok := args[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return redactAndTruncate(fmt.Sprintf("%v", v), 200)
	}
	return redactAndTruncate(s, 200)
}

func safeLogArgs(args map[string]any) string {
	if args == nil {
		return "{}"
	}
	b, err := json.Marshal(redactLogValue(args))
	if err != nil {
		return fmt.Sprintf("<marshal args: %v>", err)
	}
	return redactAndTruncate(string(b), 2000)
}

func redactLogValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, v := range x {
			if isSecretLogKey(k) {
				out[k] = "REDACTED"
				continue
			}
			out[k] = redactLogValue(v)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, v := range x {
			out[i] = redactLogValue(v)
		}
		return out
	default:
		return v
	}
}

func isSecretLogKey(k string) bool {
	k = strings.ToLower(k)
	for _, marker := range []string{"apikey", "api_key", "token", "secret", "password", "authorization"} {
		if strings.Contains(k, marker) {
			return true
		}
	}
	return false
}

func redactAndTruncate(s string, max int) string {
	s = redactKnownSecrets(s)
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func redactKnownSecrets(s string) string {
	for _, marker := range []string{"apikey=", "api_key=", "token=", "secret=", "password="} {
		start := 0
		for {
			lower := strings.ToLower(s)
			idx := strings.Index(lower[start:], marker)
			if idx < 0 {
				break
			}
			idx += start
			end := idx + len(marker)
			for end < len(s) {
				c := s[end]
				if c == '&' || c == '"' || c == '\'' || c == ' ' || c == '\n' || c == '\t' {
					break
				}
				end++
			}
			s = s[:idx+len(marker)] + "REDACTED" + s[end:]
			start = idx + len(marker) + len("REDACTED")
			if start >= len(s) {
				break
			}
		}
	}
	return s
}
