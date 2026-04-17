# AI Agent Gateway

A production-ready AI agent gateway built in Go. Sits in front of LLM calls and enforces rate limiting, prompt caching, API key authentication, and circuit breaking — while running an agentic tool-calling loop against Claude.

## Architecture

```
Client Request
      ↓
API Key Auth (PostgreSQL)
      ↓
Rate Limiter (Redis — sliding window per key)
      ↓
Circuit Breaker (auto-trip on repeated failures)
      ↓
Prompt Cache (Redis — exact match, 1hr TTL)
      ↓  cache miss
Agent Loop (Claude claude-sonnet-4-6)
      ↓
Tool Execution (calculator / URL summarizer / HN news)
      ↓
Cache Response + Return
```

## Features

- **API Key Auth** — generate per-key tokens stored in PostgreSQL, validated on every request via Bearer token
- **Rate Limiting** — sliding window counter in Redis, configurable per plan (free: 10 req/min, pro: 100 req/min)
- **Prompt Caching** — exact-match cache in Redis with 1hr TTL, repeated prompts skip the LLM entirely
- **Circuit Breaker** — trips after 5 consecutive failures, blocks requests for 30s, then half-open recovery
- **Agent Loop** — multi-turn tool-calling loop against Claude, max 5 iterations, tracks total token usage
- **Tools**
  - `calculate` — evaluates math expressions via mathjs.org API
  - `summarize_url` — fetches and extracts readable text from any URL
  - `fetch_news` — pulls top Hacker News stories with optional client-side topic filtering

## Stack

- **Go** + Gin — HTTP server and routing
- **PostgreSQL** — API key storage
- **Redis** — rate limit counters + prompt cache
- **Anthropic Claude** — LLM backend (raw HTTP, no SDK)

## Getting Started

### Prerequisites

```bash
brew install go postgresql redis
brew services start postgresql
brew services start redis
psql postgres -c "CREATE DATABASE gateway;"
```

### Setup

```bash
git clone https://github.com/binoysaha025/ai-agent-gateway
cd ai-agent-gateway
go mod download
```

Create a `.env` file in the root:

```env
PORT=8080
POSTGRES_URL=postgres://postgres:postgres@localhost:5432/gateway?sslmode=disable
REDIS_URL=redis://localhost:6379
ANTHROPIC_KEY=your_anthropic_key_here
```

### Run

```bash
go run main.go
```

## API Reference

### Generate API Key
```
POST /keys
Content-Type: application/json

{"name": "my-key", "plan": "free"}
```
```json
{"key": "abc123...", "name": "my-key", "plan": "free"}
```

### Query the Agent
```
POST /query
Authorization: Bearer <your-key>
Content-Type: application/json

{"prompt": "what is sqrt(144) + 42 * 7?"}
```
```json
{"response": "The result is 306.", "cached": false, "tokens": 1247}
```

### Health Check
```
GET /health
```

## Example Prompts

```bash
# Math
curl -X POST http://localhost:8080/query \
  -H "Authorization: Bearer <key>" \
  -H "Content-Type: application/json" \
  -d '{"prompt": "what is sqrt(144) + 42 * 7?"}'

# Latest AI news
curl -X POST http://localhost:8080/query \
  -H "Authorization: Bearer <key>" \
  -H "Content-Type: application/json" \
  -d '{"prompt": "what are the latest AI news stories?"}'

# Summarize a URL
curl -X POST http://localhost:8080/query \
  -H "Authorization: Bearer <key>" \
  -H "Content-Type: application/json" \
  -d '{"prompt": "summarize this page: https://go.dev"}'
```

## Rate Limits

| Plan | Requests/min |
|------|-------------|
| free | 10          |
| pro  | 100         |

Rate limit exceeded returns `429 Too Many Requests`. Circuit breaker open returns `503 Service Unavailable`.