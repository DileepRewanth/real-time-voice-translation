# 🌐 VoiceFlow — Real-Time English → Hindi Voice Translation Pipeline

A production-grade, real-time voice-to-voice translation system that captures live English speech, transcribes it, translates to Hindi, and speaks the result — all in near real-time.

```
🎙️ Microphone → 🧾 ASR (Web Speech API) → 🌐 Translation (Go Pipeline) → 🗣️ Hindi TTS
```

## Architecture

### Go Backend (Clean/Hexagonal Architecture)
- **Domain Layer** — Zero-dependency business models
- **Application Layer** — Use cases with caching, circuit breaker, and engine routing
- **Adapter Layer** — HTTP/WebSocket handlers, Gemini API, MyMemory API, Redis/Memory cache
- **Pipeline Engine** — Concurrent goroutine pipeline: Pre-Process → Translate → Post-Process

### React Frontend (Vite + TypeScript)
- **Custom Hooks** — `useASR`, `useTTS`, `useWebSocket`, `useAudioVisualizer`, `useSettings`
- **Real-time UI** — Live waveform visualizer, pipeline stage indicators, latency monitoring
- **Text Processing** — Filler removal, sentence boundary detection, NER protection

## Key Features

| Feature | Implementation |
|---------|---------------|
| **Dual Translation Engines** | Gemini API (context-aware AI) + MyMemory (free fallback) |
| **WebSocket Streaming** | Real-time bidirectional communication with Hub pattern |
| **Circuit Breaker** | Auto-fallback from Gemini → MyMemory on failures |
| **Redis Caching** | Translation cache with in-memory fallback |
| **Filler Removal** | Strips "uh", "um", "you know", "basically" before translation |
| **NER Protection** | Preserves brand names (Google Meet, Kubernetes, etc.) |
| **Number Localization** | "5 PM" → "शाम 5 बजे" |
| **Barge-in Support** | TTS stops when user starts speaking |
| **Graceful Shutdown** | Signal handling + in-flight request completion |
| **Docker Ready** | Multi-stage build + docker-compose with Redis |

## Quick Start

### Prerequisites
- Go 1.23+
- Node.js 20+
- Redis (optional — falls back to in-memory cache)

### 1. Backend

```bash
cd backend
cp .env.example .env  # Edit to add your Gemini API key (optional)
go run ./cmd/server/main.go
```

The server starts on `http://localhost:8080`.

### 2. Frontend

```bash
cd frontend
npm install
npm run dev
```

Open `http://localhost:5173` in Chrome/Edge.

### 3. Docker (Full Stack)

```bash
docker-compose up --build
```

### Using the Gemini Engine
The application uses the **MyMemory** free translation API by default, so it works out-of-the-box with zero configuration.

To enable the **Gemini AI** translation engine (which provides superior context-awareness and localization):
1. Obtain a free API Key from [Google AI Studio](https://aistudio.google.com/).
2. You can either:
   - Add it to your `backend/.env` file as `GEMINI_API_KEY=your_key_here` and set `DEFAULT_ENGINE=gemini`.
   - OR simply enter the API Key directly in the frontend **Settings (⚙️)** panel and switch the engine to **Gemini**.
*(Note: For enterprise environments, the codebase also natively supports GCP Service Account JSON authentication via `GOOGLE_APPLICATION_CREDENTIALS`)*

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check (server + Redis) |
| `GET` | `/health/ready` | Readiness probe |
| `POST` | `/api/v1/translate` | REST translation endpoint |
| `GET` | `/api/v1/config` | Available engines & configuration |
| `GET` | `/api/v1/ws` | WebSocket upgrade for real-time streaming |

### REST Translation Example

```bash
curl -X POST http://localhost:8080/api/v1/translate \
  -H "Content-Type: application/json" \
  -d '{"text": "Can we move the meeting to 5 PM tomorrow?", "engine": "mymemory"}'
```

Response:
```json
{
  "original_text": "Can we move the meeting to 5 PM tomorrow?",
  "processed_text": "Can we move the meeting to 5 PM tomorrow?",
  "translated_text": "क्या हम मीटिंग को कल शाम 5 बजे कर सकते हैं?",
  "engine": "mymemory",
  "cached": false,
  "latency": {
    "pre_process_ms": 1,
    "translate_ms": 342,
    "post_process_ms": 0,
    "total_ms": 343
  }
}
```

### WebSocket Message Protocol

```json
// Client → Server: Translation request
{"type": "translate", "payload": {"text": "Hello world", "engine": "gemini"}}

// Server → Client: Pipeline stage status
{"type": "status", "payload": {"stage": "translate", "status": "processing"}}

// Server → Client: Translation result
{"type": "translation", "payload": {"translated_text": "नमस्ते दुनिया", ...}}
```

## Project Structure

```
├── backend/
│   ├── cmd/server/main.go              # Entry point, DI, graceful shutdown
│   ├── internal/
│   │   ├── domain/translation.go       # Domain models (zero deps)
│   │   ├── application/
│   │   │   ├── ports/                  # Translator & Cache interfaces
│   │   │   └── usecase/translate.go    # Core orchestration + circuit breaker
│   │   ├── adapters/
│   │   │   ├── http/                   # Chi router, handlers, WebSocket hub
│   │   │   ├── translator/             # Gemini & MyMemory adapters
│   │   │   └── cache/                  # Redis & in-memory adapters
│   │   ├── pipeline/pipeline.go        # Concurrent goroutine pipeline
│   │   └── config/config.go            # Environment-based config
│   ├── Dockerfile                      # Multi-stage build
│   ├── go.mod
│   └── .env.example
├── frontend/
│   ├── src/
│   │   ├── hooks/                      # useASR, useTTS, useWebSocket, etc.
│   │   ├── components/                 # Pipeline, MicButton, Panels, etc.
│   │   ├── services/                   # Text processor, API client
│   │   └── types/                      # TypeScript interfaces
│   ├── vite.config.ts                  # Dev proxy to Go backend
│   └── package.json
├── docker-compose.yml                  # Go + Redis
└── README.md
```

## Edge Case Handling

| Edge Case | Solution |
|-----------|----------|
| Strong accents / fast speech | Web Speech API with interim results + sentence accumulation |
| Named entities mistranslated | Pre-processor marks entities; Gemini preserves them |
| Partial streaming inputs | Sentence accumulator with 600ms flush timeout |
| Fillers ("uh", "you know") | Regex-based removal before translation |
| Latency buildup | Redis cache, connection pooling, concurrent pipeline stages |
| Formal vs casual Hindi | User-configurable tone (formal/casual/auto) |
| Numbers/dates ("5 PM", "ETA") | Gemini system prompt with localization rules |
| Speech interruptions | TTS cancels on new ASR input (barge-in) |

## Tech Stack

**Backend:** Go 1.23, Chi Router, Gorilla WebSocket, go-redis, slog  
**Frontend:** React 19, TypeScript, Vite, Web Speech API, Web Audio API  
**Infrastructure:** Docker, Redis, docker-compose
