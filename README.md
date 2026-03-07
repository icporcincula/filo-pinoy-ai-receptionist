# Voice Receptionist

Local-first AI voice assistant with a browser UI. No Discord. No cloud.

```
Browser mic (16kHz PCM via AudioContext)
  │  WebSocket /ws
  ▼
Go server  :8080
  ├── VAD        energy-based, per-session, 800ms silence = end of utterance
  ├── STT   →   Speaches  :8000   (Faster-Whisper medium, GPU)
  ├── LLM   →   Ollama    :11434  (llama3:8b or any model)
  ├── TTS   →   Kokoro    :5000   (kokoro-onnx)
  └── History → Redis     :6379   (2h TTL, 20-turn window, barge-in safe)
  │  WebSocket (WAV binary + JSON events)
  ▼
Browser plays WAV via Web Audio API
```

## What runs where

| Service      | Runtime            | Start command                        |
|--------------|--------------------|--------------------------------------|
| Redis        | Docker             | `docker compose up -d`               |
| Speaches STT | Docker (GPU)       | `docker compose up -d`               |
| Ollama LLM   | Native Windows     | `ollama serve`                       |
| Kokoro TTS   | Native Python      | `python scripts/kokoro_server.py`    |
| Go server    | Native Windows/Go  | `cd server && go run main.go`        |
| UI           | Browser            | `http://localhost:8080`              |

## Setup

### 1. Infrastructure (Docker)

```bash
docker compose up -d
```

### 2. LLM (Ollama)

```bash
ollama serve
ollama pull llama3:8b
```

### 3. TTS (Kokoro)

Download model files once (run from project root):

```bash
curl -L https://github.com/thewh1teagle/kokoro-onnx/releases/download/model-files-v1.0/kokoro-v1.0.onnx -o kokoro-v1.0.onnx
curl -L https://github.com/thewh1teagle/kokoro-onnx/releases/download/model-files-v1.0/voices-v1.0.bin  -o voices-v1.0.bin
```

Then install deps and run:

```bash
pip install -r requirements.txt
python scripts/kokoro_server.py
```

### 4. Go server

```bash
cp .env.example .env        # edit if needed
source .env                 # Git Bash: set -a; . .env; set +a
cd server
go mod tidy
go run main.go
```

### 5. Open UI

```
http://localhost:8080
```

Click the mic button, speak, wait for the response.

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
└── .env.example         # Environment variables
```

## Tuning

| Variable / constant       | Default          | Notes                                  |
|---------------------------|------------------|----------------------------------------|
| `LLM_MODEL`               | `llama3:8b`      | Any model in `ollama list`             |
| `SYSTEM_PROMPT`           | see .env.example | Persona                                |
| `vadEnergyThresh` (go)    | `200`            | Raise if mic picks up background noise |
| `vadSilenceMs` (go)       | `800`            | Lower = snappier end detection         |
| `CHUNK_MS` (js)           | `100`            | PCM send interval                      |

## Barge-in

If you speak while the bot is playing back, the playback stops immediately and your new utterance takes over. The pending pipeline is cancelled via `context.CancelFunc`.
