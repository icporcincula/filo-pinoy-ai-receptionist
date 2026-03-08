# Filo — Local-First AI Receptionist for Philippine Businesses

Filo is a privacy-first, fully local AI voice receptionist built for Filipino SMEs. No cloud APIs. No data leaving your machine.

```
Browser mic (16kHz PCM via AudioContext)
  │  WebSocket /ws
  ▼
Go server  :8080
  ├── VAD        energy-based, per-session, 1200ms silence = end of utterance
  ├── STT   →   Speaches  :8000   (Faster-Whisper medium, CPU or GPU)
  ├── LLM   →   Ollama    :11434  (llama3.1:8b or any model)
  ├── TTS   →   Kokoro    :5000   (kokoro-onnx)
  └── History → Redis     :6379   (2h TTL, 20-turn window)
  │  WebSocket (WAV binary + JSON events)
  ▼
Browser plays WAV via Web Audio API
```

## What runs where

| Service      | Runtime        | Start command                     |
|--------------|----------------|-----------------------------------|
| Redis        | Docker         | `docker compose up -d`            |
| Speaches STT | Docker (CPU)   | `docker compose up -d`            |
| Ollama LLM   | Native Windows | `ollama serve`                    |
| Kokoro TTS   | Native Python  | `python scripts/kokoro_server.py` |
| Go server    | Native Windows | `go run server/main.go`           |
| UI           | Browser        | `http://localhost:8080`           |

## Setup

### 1. Infrastructure (Docker)
```bash
docker compose up -d
```

### 2. LLM (Ollama)
```bash
ollama serve
ollama pull llama3.1:8b
```

### 3. TTS (Kokoro)
Download model files once (from project root):
```bash
curl -L https://github.com/thewh1teagle/kokoro-onnx/releases/download/model-files-v1.0/kokoro-v1.0.onnx -o kokoro-v1.0.onnx
curl -L https://github.com/thewh1teagle/kokoro-onnx/releases/download/model-files-v1.0/voices-v1.0.bin  -o voices-v1.0.bin
pip install -r requirements.txt
python scripts/kokoro_server.py
```

### 4. Go server
```bash
cp .env.example .env
set -a; . .env; set +a
go run server/main.go
```

### 5. Open browser
```
http://localhost:8080
```

## Phase 1 changes (stability)

### Echo cancellation
getUserMedia now requests echoCancellation, noiseSuppression, and autoGainControl
from the browser. Prevents TTS playback from triggering false VAD hits.

### Mic muting during playback
Mic track is disabled while the bot plays audio, re-enabled when done or on
barge-in. Belt-and-suspenders on top of echo cancellation.

### WebSocket keep-alive
Go server pings every 20s, browser pongs, read deadline resets to 60s.
Prevents silent drops during long pauses between utterances.

### VAD tuning
| Constant          | Before | After | Effect                               |
|-------------------|--------|-------|--------------------------------------|
| vadSilenceMs      | 800    | 1200  | Less likely to cut off mid-sentence  |
| vadMinSpeechMs    | 200    | 100   | Catches shorter utterances           |
| vadEnergyThresh   | 200    | 100   | More sensitive to quiet voices       |

Dropped utterances are now logged: VAD: dropped short utterance (XXms)

### Whisper model keep-alive
WHISPER__MODEL_TTL: 0 in docker-compose.yml keeps the model hot.
Eliminates the 4-5s cold start after idle periods.

## Files

```
receptionist/
├── server/
│   ├── main.go          # Go WebSocket server + full pipeline
│   └── go.mod
├── ui/
│   └── index.html       # Single-file browser UI
├── scripts/
│   └── kokoro_server.py # Kokoro TTS FastAPI server
├── docker-compose.yml   # Redis + Speaches
├── requirements.txt     # Python deps
└── .env.example
```

## Tuning

| Variable              | Default       | Notes                                   |
|-----------------------|---------------|-----------------------------------------|
| LLM_MODEL             | llama3.1:8b   | Any model in ollama list                |
| SYSTEM_PROMPT         | see .env      | Persona / tone                          |
| vadEnergyThresh (go)  | 100           | Raise if background noise triggers VAD  |
| vadSilenceMs (go)     | 1200          | Lower if bot waits too long after you   |
| vadMinSpeechMs (go)   | 100           | Lower if short phrases are missed       |

## Roadmap

Phase 2 — Intelligence
- Qdrant RAG (business docs: FAQs, pricing, hours)
- Intent classification (book / inquire / transfer)
- n8n webhook for appointment booking → Google Calendar
- Persona config via .env (business name, language, tone)

Phase 3 — Deployment
- Dockerfile for Go server + Kokoro
- GPU Speaches (NVIDIA CDI via WSL2)
- Nginx + HTTPS (required for mic on non-localhost)
- Multi-tenant persona config

Phase 4 — WhatsApp Channel
- Meta Cloud API webhook → Filo pipeline
- Receive voice notes → STT → LLM → reply as voice note or text
- OpenClaw as WhatsApp agent orchestration layer
- n8n for appointment confirmations → Google Calendar → SMS
- Filo lives in the business owner's WhatsApp like a staff member
