# AI Agent Gateway

A production-ready AI agent gateway built in Go. Sits in front of LLM calls and enforces rate limiting, prompt caching, API key authentication, and circuit breaking — while running an agentic RAG pipeline with ensemble self-reflection against Claude.

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
Tool Execution — Claude decides which tools to call:
  ├── calculate       → math expression via mathjs API
  ├── summarize_url   → fetch + extract text from any URL
  ├── fetch_news      → Hacker News top stories with topic filtering
  └── search_knowledge → semantic search over pgvector knowledge base
      ↓
Ensemble Critic (3 parallel goroutines)
  ├── Factuality critic   → checks for hallucinations
  ├── Completeness critic → checks if question fully answered
  └── Groundedness critic → checks if claims are supported
      ↓
Majority vote (2/3) → PASS: return response | FAIL: retry with feedback
      ↓
Cache Response + Return
```

## Features

**Infrastructure:**
- **API Key Auth** — generate per-key tokens stored in PostgreSQL, validated on every request via Bearer token
- **Rate Limiting** — sliding window counter in Redis, configurable per plan (free: 10 req/min, pro: 100 req/min)
- **Prompt Caching** — exact-match cache in Redis with 1hr TTL, repeated prompts skip the LLM entirely
- **Circuit Breaker** — trips after 5 consecutive failures, blocks for 30s, half-open recovery with mutex-protected state

**Agentic AI:**
- **Agentic RAG** — Claude autonomously decides when to call `search_knowledge`, retrieves semantically similar chunks from pgvector using cosine similarity, reasons over retrieved context
- **Auto-chunking** — documents are split on sentence boundaries into ~500 char chunks before embedding, preserving semantic integrity
- **Ensemble Critic** — 3 specialized critics (factuality, completeness, groundedness) run as parallel Go goroutines, majority vote determines response quality
- **Self-Reflection** — failed critic evaluations trigger a retry with critic feedback appended to the context window

## Stack

- **Go** + Gin — HTTP server, routing, middleware
- **PostgreSQL** + pgvector — API key storage + vector similarity search
- **Redis** — rate limit counters + prompt cache
- **Voyage AI** — text embeddings (1024-dim, voyage-3)
- **Anthropic Claude** — LLM backend (raw HTTP, no SDK)

## Getting Started

### Prerequisites

```bash
brew install go postgresql@17 redis pgvector
brew services start postgresql@17
brew services start redis
psql postgres -U $USER -c "CREATE ROLE postgres WITH SUPERUSER LOGIN PASSWORD 'postgres123';"
psql postgres -c "CREATE DATABASE gateway;"
psql postgres -d gateway -c "CREATE EXTENSION IF NOT EXISTS vector;"
```

### Setup

```bash
git clone https://github.com/binoysaha025/ai-agent-gateway
cd ai-agent-gateway
go mod download
```

Create `.env` in the root:

```env
PORT=8080
POSTGRES_URL=postgres://postgres:postgres123@localhost:5432/gateway?sslmode=disable
REDIS_URL=redis://localhost:6379
ANTHROPIC_KEY=your_anthropic_key
VOYAGE_API_KEY=your_voyage_key
```

Get a free Voyage AI key at [voyageai.com](https://voyageai.com).

### Run

```bash
go run main.go
```

## API Reference

### Generate API Key
```
POST /keys
{"name": "my-key", "plan": "free"}
```
```json
{"key": "abc123...", "name": "my-key", "plan": "free"}
```

### Embed a Document
```
POST /embed
{"content": "your document text here", "metadata": "source-name"}
```
```json
{"message": "document embedded and stored", "chunks": 3}
```
Documents are auto-chunked on sentence boundaries before embedding.

### Query the Agent
```
POST /query
Authorization: Bearer <your-key>
{"prompt": "your question here"}
```
```json
{"response": "...", "cached": false, "tokens": 1247}
```

### Health Check
```
GET /health
```

## Example Prompts

```bash
KEY="your-api-key"

# Math
curl -X POST http://localhost:8080/query \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{"prompt": "what is sqrt(144) + 42 * 7?"}'

# RAG retrieval from knowledge base
curl -X POST http://localhost:8080/query \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{"prompt": "what are the key features of Go?"}'

# Latest AI news
curl -X POST http://localhost:8080/query \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{"prompt": "what are the latest AI news stories?"}'

# Summarize a URL
curl -X POST http://localhost:8080/query \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{"prompt": "summarize this page: https://go.dev"}'
```

## Rate Limits

| Plan | Requests/min |
|------|-------------|
| free | 10          |
| pro  | 100         |

- Rate limit exceeded → `429 Too Many Requests`
- Circuit breaker open → `503 Service Unavailable`
- Invalid/missing key → `401 Unauthorized`

## Project Structure

```
ai-agent-gateway/
├── main.go              # entry point, routing
├── config/config.go     # env var loading
├── db/postgres.go       # PostgreSQL connection
├── cache/
│   ├── redis.go         # Redis connection
│   └── prompt.go        # prompt cache (get/set)
├── middleware/
│   ├── auth.go          # API key validation
│   ├── ratelimit.go     # sliding window rate limiter
│   └── circuitbreaker.go # circuit breaker with half-open recovery
├── handlers/routes.go   # HTTP handlers + auto-chunking
├── models/apikey.go     # DB schema + queries
├── agent/
│   ├── agent.go         # agent loop + tool execution
│   └── critic.go        # ensemble critic + self-reflection
└── tools/
    ├── tools.go         # calculator, URL summarizer, news fetcher
    ├── rag.go           # pgvector similarity search
    └── embed.go         # Voyage AI embedding generation
```
