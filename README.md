# golang-local-fmp-mcp

An FMP MCP server in Go. It exposes a JSON-RPC / MCP-style endpoint wrapping the
[Financial Modeling Prep (FMP)](https://financialmodelingprep.com) API. FMP is
available at https://financialmodelingprep.com.

The server listens at:

```
http://host:port/mcp
```

Default host/port: `0.0.0.0:8086`.

## Tools

The server registers one tool per FMP category. Each tool takes an `endpoint`
argument (an enum selecting the FMP sub-action) and a small set of shared
parameters. The categories mirror the upstream FMP MCP one-to-one:

- `ESG`
- `Fundraisers`
- `analyst`
- `calendar`
- `chart`
- `commitmentOfTraders`
- `commodity`
- `company`
- `crypto`
- `directory`
- `discountedCashFlow`
- `earningsTranscript`
- `economics`
- `etfAndMutualFunds`
- `forex`
- `form13F`
- `indexes`
- `insiderTrades`
- `marketHours`
- `marketPerformance`
- `news`
- `quote`
- `search`
- `secFilings`
- `senate`
- `statements`
- `technicalIndicators`

The full list of `endpoint` values per tool comes back from `tools/list`.

## Setup

```
go mod tidy
cp .env.example .env
# edit .env and set FMP_API_KEY
./go.sh
```

Stop the server with Ctrl+C. The Go process listens for SIGINT/SIGTERM and
shuts the HTTP listener down gracefully.

## Config

`.env` (gitignored):

```
FMP_API_KEY=your-fmp-api-key-here
```

`config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8086
  mcp_path: "/mcp"

fmp:
  base_url: "https://financialmodelingprep.com"
  api_path: "/stable"
  timeout_seconds: 30
  user_agent: "golang-local-fmp-mcp/0.1"
```

`fmp.api_path` defaults to FMP's `/stable` namespace. If a particular endpoint
lives elsewhere (`/api/v3`, `/api/v4`), edit either `api_path` for the whole
server or the per-endpoint override map in
`internal/tools/handlers.go`.

## JSON-RPC requests

Initialize:

```bash
curl -s http://127.0.0.1:8086/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | jq
```

List tools:

```bash
curl -s http://127.0.0.1:8086/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | jq
```

Stock quote:

```bash
curl -s http://127.0.0.1:8086/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"quote","arguments":{"endpoint":"quote","symbol":"AAPL"}}}' | jq
```

Stock search:

```bash
curl -s http://127.0.0.1:8086/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"search","arguments":{"endpoint":"search-symbol","query":"AAPL"}}}' | jq
```

Income statement (annual, last 5):

```bash
curl -s http://127.0.0.1:8086/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"statements","arguments":{"endpoint":"income-statement","symbol":"AAPL","period":"annual","limit":5}}}' | jq
```

EOD chart:

```bash
curl -s http://127.0.0.1:8086/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"chart","arguments":{"endpoint":"historical-price-eod-light","symbol":"AAPL","from_date":"2026-05-01","to_date":"2026-05-25"}}}' | jq
```

Intraday 5-minute bars:

```bash
curl -s http://127.0.0.1:8086/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"chart","arguments":{"endpoint":"intraday-5-min","symbol":"AAPL"}}}' | jq
```

A 14-period RSI on 1-hour bars:

```bash
curl -s http://127.0.0.1:8086/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"technicalIndicators","arguments":{"endpoint":"relative-strength-index","symbol":"AAPL","periodLength":14,"timeframe":"1hour"}}}' | jq
```

General market news:

```bash
curl -s http://127.0.0.1:8086/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"news","arguments":{"endpoint":"general-news","limit":10}}}' | jq
```

Health check:

```bash
curl -s http://127.0.0.1:8086/healthz
# ok
```

## Periodic Maintenance

FMP can rename or move API paths. Every so often, especially before a release,
run a small smoke test and keep the server logs.

```bash
./go.sh 2>&1 | tee fmp-mcp.log
```

In another terminal:

```bash
examples/test_client.sh
```

Then test one or two endpoints you rely on most, for example:

```bash
curl -s http://127.0.0.1:8086/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":20,"method":"tools/call","params":{"name":"company","arguments":{"endpoint":"market-cap","symbol":"NVDA"}}}' | jq

curl -s http://127.0.0.1:8086/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":21,"method":"tools/call","params":{"name":"news","arguments":{"endpoint":"search-stock-news","symbols":["NVDA"],"limit":1}}}' | jq
```

If anything fails, paste the relevant `tool_call_failed` lines from
`fmp-mcp.log` into an LLM along with the tool request that failed. Those lines
include the JSON-RPC id, tool, endpoint, sanitized arguments, duration, and the
redacted FMP URL/body when FMP returned an upstream error.

Failure logs should always be detailed enough to fix the issue from the log
alone: keep the tool name, endpoint, sanitized arguments, redacted upstream URL,
HTTP status/body, and duration in the log output. Do not remove that context
when changing logging.

Useful prompt:

```text
This is a Go FMP MCP server. Fix the endpoint mapping or argument forwarding
based on these logs. Do not expose API keys.

<paste tool_call_failed lines here>
<paste the JSON-RPC request that failed here>
```

## Layout

```
cmd/golang-local-fmp-mcp/main.go # entrypoint, signal handling
internal/config/                # config.yaml + .env loader
internal/fmp/                   # HTTP client for FMP API
internal/server/                # JSON-RPC handler at /mcp
internal/tools/                 # tool registry + 27 category handlers
examples/test_client.sh         # smoke-test script
config.yaml
.env.example
go.sh
```

## Notes

- `.env` is gitignored; never commit your `FMP_API_KEY`.
- The API key is appended to every outbound request automatically and is
  redacted from any error messages that surface a URL.
- Each tool call logs its method name and duration. Failed upstream FMP request
  URLs are logged only after API-key redaction.
- If a tool fails, copy the `tool_call_failed` log line back into an issue or
  chat. It includes the JSON-RPC id, tool, endpoint, sanitized arguments,
  duration, and the redacted upstream FMP error URL/body when available.
- Failure logs are intentionally verbose enough that endpoint mapping or
  argument-forwarding bugs can be fixed from pasted log output without exposing
  the API key.
- The `endpoint` -> URL path mapping in `internal/tools/handlers.go` reflects
  FMP's `/stable` API at the time of writing. If FMP renames paths, edit the
  `overrides` map for the affected tool.
