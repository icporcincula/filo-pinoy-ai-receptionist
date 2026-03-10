package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// ── Config ────────────────────────────────────────────────────────────────────

type Config struct {
	Port        string
	SpeachesURL string
	OllamaURL   string
	KokoroURL   string
	RedisAddr   string
	LLMModel    string
	SystemPrompt string
	TENVADMode  int
	TurnDetectionBaseURL string
	TurnDetectionAPIKey  string
	TurnDetectionModel   string
	TurnDetectionTemperature float64
	TurnDetectionTopP   float64
	TurnDetectionThresholdMs int
}

func loadConfig() Config {
	get := func(key, def string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return def
	}
	
	// Helper functions to get integer and float values from env
	getInt := func(key string, def int) int {
		if v := os.Getenv(key); v != "" {
			if val, err := strconv.Atoi(v); err == nil {
				return val
			}
		}
		return def
	}
	
	getFloat := func(key string, def float64) float64 {
		if v := os.Getenv(key); v != "" {
			if val, err := strconv.ParseFloat(v, 64); err == nil {
				return val
			}
		}
	return def
	}
	
	return Config{
		Port:        get("PORT", "8080"),
		SpeachesURL: get("SPEACHES_URL", "http://localhost:8000"),
		OllamaURL:   get("OLLAMA_URL", "http://localhost:11434"),
		KokoroURL:   get("KOKORO_URL", "http://localhost:5000"),
		RedisAddr:   get("REDIS_ADDR", "localhost:6379"),
		LLMModel:    get("LLM_MODEL", "llama3.1:8b"),
		SystemPrompt: get("SYSTEM_PROMPT",
			"You are a helpful, concise voice assistant. Keep responses short and conversational. Avoid markdown formatting."),
		TENVADMode: getInt("TEN_VAD_MODE", 2),
		TurnDetectionBaseURL: get("TURN_DETECTION_BASE_URL", "http://localhost:8000/v1"),
		TurnDetectionAPIKey:  get("TURN_DETECTION_API_KEY", "your-turn-detection-api-key"),
		TurnDetectionModel:   get("TURN_DETECTION_MODEL", "turn-detection-model"),
		TurnDetectionTemperature: getFloat("TURN_DETECTION_TEMPERATURE", 0.1),
		TurnDetectionTopP:   getFloat("TURN_DETECTION_TOP_P", 0.1),
		TurnDetectionThresholdMs: getInt("TURN_DETECTION_THRESHOLD_MS", 500),
	}
}

// ── WebSocket messages ────────────────────────────────────────────────────────

// Client → Server
type InboundMsg struct {
	Type string `json:"type"`
	// type: "audio"  — PCM chunk as raw binary (separate ws message)
	// type: "config" — future: persona, language
}

// Server → Client
type OutboundMsg struct {
	Type       string `json:"type"`
	Text       string `json:"text,omitempty"`
	AudioB64   []byte `json:"audio,omitempty"` // WAV bytes, client decodes
	Transcript string `json:"transcript,omitempty"`
	Error      string `json:"error,omitempty"`
}


// ── Session ───────────────────────────────────────────────────────────────────

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type session struct {
	id     string
	cfg    Config
	conn   *websocket.Conn
	rdb    *redis.Client
	send   chan []byte
	vad    *TenVAD
	turnDetector *TurnDetector

	// pipeline cancellation: cancel current TTS/LLM if user starts speaking
	cancelMu  sync.Mutex
	cancelFn  context.CancelFunc
}

func newSession(id string, conn *websocket.Conn, cfg Config, rdb *redis.Client) *session {
	s := &session{
		id:   id,
		cfg:  cfg,
		conn: conn,
		rdb:  rdb,
		send: make(chan []byte, 64),
	}
	s.vad = NewTenVAD(s.onUtterance)
	s.vad.SetMode(cfg.TENVADMode) // Set the VAD mode from config
	
	// Initialize turn detector with config values
	turnConfig := &TurnDetectionConfig{
		BaseURL:     cfg.TurnDetectionBaseURL,
		APIKey:      cfg.TurnDetectionAPIKey,
		Model:       cfg.TurnDetectionModel,
		Temperature: cfg.TurnDetectionTemperature,
		TopP:        cfg.TurnDetectionTopP,
		ThresholdMs: cfg.TurnDetectionThresholdMs,
	}
	s.turnDetector = NewTurnDetector(turnConfig)
	
	return s
}

// onUtterance is called by VAD in a separate goroutine when speech ends.
func (s *session) onUtterance(pcm []int16) {
	// Cancel any in-flight response (barge-in)
	s.cancelMu.Lock()
	if s.cancelFn != nil {
		s.cancelFn()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	s.cancelFn = cancel
	s.cancelMu.Unlock()
	defer cancel()

	// Signal barge-in to client
	s.sendJSON(OutboundMsg{Type: "barge_in"})

	// 1. STT
	transcript, err := s.transcribe(ctx, pcm)
	if err != nil {
		log.Printf("[%s] STT error: %v", s.id, err)
		s.sendJSON(OutboundMsg{Type: "error", Error: "transcription failed"})
		return
	}
	if transcript == "" {
		return
	}
	s.sendJSON(OutboundMsg{Type: "transcript", Transcript: transcript})
	log.Printf("[%s] USER: %s", s.id, transcript)

	// 2. Turn Detection - Check if user has finished speaking
	turnCtx, turnCancel := context.WithTimeout(ctx, 5*time.Second)
	turnState, err := s.turnDetector.Evaluate(turnCtx, transcript)
	turnCancel()
	
	if err != nil {
		log.Printf("[%s] Turn detection error: %v", s.id, err)
		// Continue with processing even if turn detection fails
	} else {
		log.Printf("[%s] Turn detection state: %s", s.id, turnState.String())
		
		// If turn detection indicates user is not finished, skip processing
		if turnState == TurnStateUnfinished {
			log.Printf("[%s] User not finished speaking, skipping processing", s.id)
			return
		}
	}

	// 3. History
	history, _ := s.getHistory(ctx)
	history = append(history, message{Role: "user", Content: transcript})

	// 4. LLM
	reply, err := s.chat(ctx, history)
	if err != nil {
		log.Printf("[%s] LLM error: %v", s.id, err)
		s.sendJSON(OutboundMsg{Type: "error", Error: "LLM failed"})
		return
	}
	log.Printf("[%s] BOT: %s", s.id, reply)
	s.sendJSON(OutboundMsg{Type: "reply_text", Text: reply})

	// 5. Save history
	history = append(history, message{Role: "assistant", Content: reply})
	s.saveHistory(ctx, history)

	// 6. TTS
	wav, err := s.synthesize(ctx, reply)
	if err != nil {
		log.Printf("[%s] TTS error: %v", s.id, err)
		// Still delivered text, just no audio
		return
	}

	s.sendBinary(wav)
}

// writePump drains send channel → WebSocket
func (s *session) writePump() {
	for data := range s.send {
		if err := s.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
			log.Printf("[%s] write error: %v", s.id, err)
			return
		}
	}
}

func (s *session) sendJSON(msg OutboundMsg) {
	b, _ := json.Marshal(msg)
	// prefix 0x00 = JSON, 0x01 = raw audio
	framed := append([]byte{0x00}, b...)
	select {
	case s.send <- framed:
	default:
	}
}

func (s *session) sendBinary(wav []byte) {
	// prefix 0x01 = audio WAV
	framed := append([]byte{0x01}, wav...)
	select {
	case s.send <- framed:
	default:
	}
}

// readPump reads from WebSocket and feeds VAD.
// Binary messages with no prefix = raw PCM s16le 16kHz mono.
func (s *session) readPump() {
	defer close(s.send)
	for {
		mt, data, err := s.conn.ReadMessage()
		if err != nil {
			break
		}
		if mt == websocket.BinaryMessage {
			// raw PCM from browser
			samples := bytesToInt16(data)
			s.vad.feed(samples)
		}
		// text messages reserved for future config
	}
}

// ── STT ───────────────────────────────────────────────────────────────────────

func (s *session) transcribe(ctx context.Context, pcm []int16) (string, error) {
	// Build WAV in memory
	wav := pcmToWAV(pcm, 16000, 1)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", "audio.wav")
	if err != nil {
		return "", err
	}
	fw.Write(wav)
	mw.WriteField("model", "Systran/faster-whisper-medium")
	mw.WriteField("language", "en")
	mw.WriteField("response_format", "json")
	mw.Close()

	req, _ := http.NewRequestWithContext(ctx, "POST",
		s.cfg.SpeachesURL+"/v1/audio/transcriptions", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Text string `json:"text"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Text, nil
}

// ── LLM ───────────────────────────────────────────────────────────────────────

func (s *session) chat(ctx context.Context, history []message) (string, error) {
	messages := []map[string]string{
		{"role": "system", "content": s.cfg.SystemPrompt},
	}
	for _, m := range history {
		messages = append(messages, map[string]string{
			"role": m.Role, "content": m.Content,
		})
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"model":    s.cfg.LLMModel,
		"messages": messages,
		"stream":   false,
	})

	req, _ := http.NewRequestWithContext(ctx, "POST",
		s.cfg.OllamaURL+"/api/chat", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	rawBody, _ := io.ReadAll(resp.Body)
	log.Printf("[%s] LLM raw: %s", s.id, string(rawBody))
	var result struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	json.Unmarshal(rawBody, &result)
	return result.Message.Content, nil
}

// ── TTS ───────────────────────────────────────────────────────────────────────

func (s *session) synthesize(ctx context.Context, text string) ([]byte, error) {
	payload, _ := json.Marshal(map[string]interface{}{
		"input": text,
		"voice": "af_sky",
		"response_format": "wav",
	})

	req, _ := http.NewRequestWithContext(ctx, "POST",
		s.cfg.KokoroURL+"/v1/audio/speech", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// ── History (Redis) ───────────────────────────────────────────────────────────

const historyKey = "receptionist:history:"
const maxHistory = 20

func (s *session) getHistory(ctx context.Context) ([]message, error) {
	key := historyKey + s.id
	data, err := s.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var h []message
	json.Unmarshal(data, &h)
	return h, nil
}

func (s *session) saveHistory(ctx context.Context, h []message) {
	if len(h) > maxHistory {
		h = h[len(h)-maxHistory:]
	}
	data, _ := json.Marshal(h)
	s.rdb.Set(ctx, historyKey+s.id, data, 2*time.Hour)
}

// ── PCM / WAV helpers ─────────────────────────────────────────────────────────

func bytesToInt16(b []byte) []int16 {
	out := make([]int16, len(b)/2)
	for i := range out {
		out[i] = int16(binary.LittleEndian.Uint16(b[i*2:]))
	}
	return out
}

func pcmToWAV(pcm []int16, sampleRate, channels int) []byte {
	numSamples := len(pcm)
	dataSize := numSamples * 2
	var buf bytes.Buffer
	// RIFF header
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(36+dataSize))
	buf.WriteString("WAVE")
	// fmt chunk
	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16))
	binary.Write(&buf, binary.LittleEndian, uint16(1)) // PCM
	binary.Write(&buf, binary.LittleEndian, uint16(channels))
	binary.Write(&buf, binary.LittleEndian, uint32(sampleRate))
	binary.Write(&buf, binary.LittleEndian, uint32(sampleRate*channels*2))
	binary.Write(&buf, binary.LittleEndian, uint16(channels*2))
	binary.Write(&buf, binary.LittleEndian, uint16(16))
	// data chunk
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, uint32(dataSize))
	for _, s := range pcm {
		binary.Write(&buf, binary.LittleEndian, s)
	}
	return buf.Bytes()
}

// ── HTTP / WS server ──────────────────────────────────────────────────────────

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	cfg := loadConfig()

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("WARNING: Redis unavailable (%v) — history disabled", err)
	}

	// Serve static UI
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "ui/index.html")
	})

	// Health
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"ok":true}`)
	})

	// WebSocket
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade: %v", err)
			return
		}
		sessionID := fmt.Sprintf("%d", time.Now().UnixNano())
		log.Printf("new session: %s", sessionID)

		s := newSession(sessionID, conn, cfg, rdb)
		go s.writePump()
		s.readPump() // blocks until disconnect
		log.Printf("session ended: %s", sessionID)
	})

	log.Printf("Receptionist server listening on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}
