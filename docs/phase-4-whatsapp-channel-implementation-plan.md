# Phase 4 — WhatsApp Channel Implementation Plan

## Overview

Phase 4 extends Filo beyond the web interface to integrate with WhatsApp, enabling businesses to serve customers through the popular messaging platform. This phase transforms Filo into a multi-channel business assistant accessible via both web and WhatsApp.

## Current State (Post-Phase 3)

After Phase 3, Filo will have:
- ✅ Production-ready containerized deployment
- ✅ GPU acceleration for optimal performance
- ✅ HTTPS security and multi-tenant support
- ✅ Intelligent features (RAG, intent classification, workflows)
- ✅ Stable, scalable infrastructure

## Phase 4 Goals

Transform Filo into a multi-channel business assistant with:
- **WhatsApp Integration**: Meta Cloud API integration for WhatsApp messaging
- **Voice Note Processing**: Convert WhatsApp voice notes to text and responses
- **OpenClaw Orchestration**: Use OpenClaw for WhatsApp agent management
- **n8n Workflow Extensions**: Extend workflows for WhatsApp-specific scenarios
- **Cross-Channel Consistency**: Maintain consistent experience across web and WhatsApp

## Implementation Components

### 1. Meta Cloud API Integration

**Purpose**: Connect Filo to WhatsApp Business API for message handling

**Implementation**:
```go
type WhatsAppClient struct {
    baseURL    string
    apiKey     string
    phoneNumberID string
    httpClient *http.Client
}

type WhatsAppMessage struct {
    From        string `json:"from"`
    To          string `json:"to"`
    Type        string `json:"type"`
    Text        *TextMessage `json:"text,omitempty"`
    Audio       *AudioMessage `json:"audio,omitempty"`
    Context     *MessageContext `json:"context,omitempty"`
}

func (wc *WhatsAppClient) SendMessage(msg WhatsAppMessage) error {
    payload, _ := json.Marshal(map[string]interface{}{
        "messaging_product": "whatsapp",
        "to": msg.From,
        "type": msg.Type,
        msg.Type: msg,
    })
    
    req, _ := http.NewRequest("POST", wc.baseURL+"/messages", bytes.NewBuffer(payload))
    req.Header.Set("Authorization", "Bearer "+wc.apiKey)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := wc.httpClient.Do(req)
    // Handle response...
}
```

**Integration Points**:
- Webhook endpoint for incoming WhatsApp messages
- Message routing to Filo pipeline
- Response handling for text and voice notes
- Message status tracking

**Files to Create**:
- `server/whatsapp.go` - WhatsApp API client
- `server/whatsapp_webhook.go` - Webhook handler
- Update `server/main.go` for WhatsApp message routing

### 2. Voice Note Processing

**Purpose**: Convert WhatsApp voice notes to text and process through Filo pipeline

**Implementation**:
```go
func (wc *WhatsAppClient) ProcessVoiceNote(voiceURL string, from string) error {
    // Download voice note
    audioData, err := wc.downloadAudio(voiceURL)
    if err != nil {
        return err
    }
    
    // Convert to 16kHz PCM for STT
    pcmData, err := wc.convertToPCM(audioData)
    if err != nil {
        return err
    }
    
    // Process through Filo pipeline
    transcript, err := wc.transcribeAudio(pcmData)
    if err != nil {
        return err
    }
    
    // Route to LLM with WhatsApp context
    response, err := wc.processThroughFilo(transcript, from, "whatsapp")
    if err != nil {
        return err
    }
    
    // Send response back to WhatsApp
    return wc.sendTextMessage(from, response)
}
```

**Integration Points**:
- Audio format conversion (WhatsApp → 16kHz PCM)
- Integration with existing STT pipeline
- Context preservation for conversation state
- Response format adaptation for WhatsApp

**Files to Create/Modify**:
- `server/audio_converter.go` - Audio format conversion
- Update STT integration for WhatsApp audio
- Add WhatsApp-specific response formatting

### 3. OpenClaw WhatsApp Agent Orchestration

**Purpose**: Use OpenClaw as the orchestration layer for WhatsApp agent management

**Implementation**:
```yaml
# openclaw-config.yml
agents:
  whatsapp-receptionist:
    type: whatsapp
    config:
      phone_number: "+1234567890"
      business_name: "Your Business"
      webhook_url: "https://your-domain.com/whatsapp/webhook"
    
    pipeline:
      - name: audio_processing
        type: stt
        config:
          model: "medium"
          language: "filipino"
      
      - name: intent_classification
        type: intent
        config:
          model: "intent_model.onnx"
      
      - name: response_generation
        type: llm
        config:
          model: "llama3.1:8b"
          context: "whatsapp"
      
      - name: response_delivery
        type: whatsapp
        config:
          format: "text_or_voice"
```

**Integration Points**:
- OpenClaw configuration for WhatsApp agents
- Agent lifecycle management
- Message routing and load balancing
- Monitoring and health checks

**Files to Create**:
- `openclaw/whatsapp-agent.yml` - OpenClaw configuration
- `scripts/deploy_whatsapp_agent.sh` - Agent deployment script
- Update OpenClaw integration

### 4. n8n WhatsApp Workflows

**Purpose**: Extend n8n workflows for WhatsApp-specific business scenarios

**Implementation**:
```json
{
  "name": "WhatsApp Appointment Booking",
  "nodes": [
    {
      "name": "Webhook Trigger",
      "type": "n8n-nodes-base.webhook",
      "webhookId": "whatsapp-appointment",
      "httpMethod": "POST"
    },
    {
      "name": "Parse WhatsApp Message",
      "type": "n8n-nodes-base.function",
      "functionCode": "return [{\n  phoneNumber: $json.from,\n  message: $json.text.body,\n  timestamp: $json.timestamp\n}];"
    },
    {
      "name": "Intent Classification",
      "type": "n8n-nodes-base.httpRequest",
      "url": "http://filo-server:8080/intent",
      "method": "POST",
      "jsonParameters": true,
      "bodyParametersJson": "{\"text\": \"{{$json[\"message\"]}}\"}"
    },
    {
      "name": "Book Appointment",
      "type": "n8n-nodes-base.googleCalendar",
      "operation": "create",
      "calendarId": "primary",
      "summary": "Customer Appointment",
      "description": "{{$json[\"message\"]}}",
      "start": "{{$json[\"timestamp\"]}}",
      "end": "{{$json[\"timestamp\"] + 3600}}"
    },
    {
      "name": "Send Confirmation",
      "type": "n8n-nodes-base.httpRequest",
      "url": "http://filo-server:8080/whatsapp/send",
      "method": "POST",
      "jsonParameters": true,
      "bodyParametersJson": {
        "to": "{{$json[\"phoneNumber\"]}}",
        "message": "Your appointment has been booked!"
      }
    }
  ]
}
```

**Integration Points**:
- WhatsApp-specific workflow templates
- Integration with Google Calendar
- SMS notification workflows
- Customer relationship management

**Files to Create**:
- `n8n/workflows/whatsapp-appointment.json` - Appointment booking workflow
- `n8n/workflows/whatsapp-inquiry.json` - Inquiry handling workflow
- `n8n/workflows/whatsapp-order.json` - Order processing workflow

### 5. Cross-Channel Consistency

**Purpose**: Ensure consistent experience and data across web and WhatsApp channels

**Implementation**:
```go
type ChannelContext struct {
    Channel     string    `json:"channel"`     // "web" or "whatsapp"
    SessionID   string    `json:"session_id"`
    UserID      string    `json:"user_id"`
    TenantID    string    `json:"tenant_id"`
    LastActive  time.Time `json:"last_active"`
}

type CrossChannelManager struct {
    sessions map[string]*ChannelContext
    mutex    sync.RWMutex
}

func (ccm *CrossChannelManager) RouteMessage(msg interface{}, channel string) (*ChannelContext, error) {
    // Determine if this is a continuation of existing session
    // or new session based on user identifier
    // Route to appropriate tenant and persona
    // Maintain conversation context across channels
}
```

**Integration Points**:
- Unified session management
- Cross-channel conversation history
- Consistent persona and knowledge base
- Channel-specific optimizations

**Files to Create**:
- `server/cross_channel.go` - Cross-channel management
- Update session management for multi-channel support
- Add channel-aware response generation

## Implementation Timeline

### Week 1: WhatsApp API Foundation
- [ ] Set up Meta Cloud API integration
- [ ] Create webhook endpoint for message handling
- [ ] Implement basic message sending/receiving
- [ ] Test WhatsApp API connectivity

### Week 2: Voice Note Processing
- [ ] Implement audio download and conversion
- [ ] Integrate with existing STT pipeline
- [ ] Add WhatsApp-specific response formatting
- [ ] Test voice note to text conversion

### Week 3: OpenClaw Integration
- [ ] Configure OpenClaw for WhatsApp agents
- [ ] Set up agent orchestration
- [ ] Implement monitoring and health checks
- [ ] Test agent deployment and management

### Week 4: n8n Workflows & Cross-Channel
- [ ] Create WhatsApp-specific n8n workflows
- [ ] Integrate with Google Calendar and SMS
- [ ] Implement cross-channel session management
- [ ] Test end-to-end multi-channel scenarios

## Technical Architecture

### Multi-Channel Architecture
```
Web Channel                    WhatsApp Channel
     │                              │
     ▼                              ▼
WebSocket Server              Meta Cloud API
     │                              │
     ▼                              ▼
Filo Core Engine ←── Cross-Channel Manager
     │
     ├── STT Pipeline
     ├── Intent Classification
     ├── RAG System
     ├── LLM Processing
     ├── Workflow Router
     └── TTS/Response Generation
     │
     ▼                              ▼
Web UI                        WhatsApp Messages
```

### WhatsApp Message Flow
```
WhatsApp User
     │
     ▼
Voice Note / Text Message
     │
     ▼
Meta Cloud API Webhook
     │
     ▼
Audio Processing (if voice note)
     │
     ▼
Filo Pipeline (STT → Intent → LLM → Response)
     │
     ▼
Response (Text or Voice Note)
     │
     ▼
WhatsApp User
```

## Dependencies and Requirements

### New Dependencies
- **Meta Cloud API**: WhatsApp Business API access
- **OpenClaw**: Agent orchestration platform
- **n8n**: Workflow automation (extended)
- **Audio Processing Libraries**: For WhatsApp audio format conversion

### Configuration Requirements
- Meta Cloud API credentials and phone number
- OpenClaw deployment and configuration
- n8n workflow definitions
- Cross-channel session storage

## Success Criteria

### Functional Requirements
- [ ] WhatsApp message sending/receiving works reliably
- [ ] Voice note processing achieves >90% accuracy
- [ ] Cross-channel session continuity maintained
- [ ] n8n workflows handle business scenarios successfully

### Performance Requirements
- [ ] WhatsApp message response time <3 seconds
- [ ] Voice note processing adds <2 seconds latency
- [ ] Cross-channel context switching <100ms
- [ ] System handles 100+ concurrent WhatsApp sessions

### Business Requirements
- [ ] Support for common business interactions (inquiries, bookings, orders)
- [ ] Integration with existing business tools (Google Calendar, SMS)
- [ ] Consistent experience across web and WhatsApp
- [ ] Multi-tenant support for WhatsApp channels

## Risk Mitigation

### Technical Risks
- **API Rate Limits**: Implement proper rate limiting and queuing
- **Audio Quality**: Handle various audio formats and quality levels
- **Message Delivery**: Implement retry mechanisms for failed messages
- **Cross-Channel Complexity**: Thorough testing of session management

### Business Risks
- **User Adoption**: Provide clear documentation and onboarding
- **Message Costs**: Monitor and optimize WhatsApp API usage costs
- **Regulatory Compliance**: Ensure compliance with messaging regulations
- **Business Continuity**: Implement fallback mechanisms

## Configuration Management

### Environment Variables
```env
# WhatsApp Configuration
WHATSAPP_ENABLED=true
WHATSAPP_PHONE_NUMBER="+1234567890"
WHATSAPP_API_KEY=your_meta_cloud_api_key
WHATSAPP_WEBHOOK_VERIFY_TOKEN=your_verify_token

# OpenClaw Configuration
OPENCLAW_ENABLED=true
OPENCLAW_URL=http://openclaw:8080
OPENCLAW_API_KEY=your_openclaw_key

# Cross-Channel Configuration
CROSS_CHANNEL_ENABLED=true
SESSION_STORE_TYPE=redis
SESSION_TTL=3600
```

### Deployment Scripts
```bash
# scripts/deploy_whatsapp.sh
#!/bin/bash
set -e

echo "Setting up WhatsApp integration..."
./scripts/setup_whatsapp_api.sh

echo "Deploying OpenClaw agents..."
docker-compose -f docker-compose.whatsapp.yml up -d

echo "Configuring n8n workflows..."
./scripts/setup_whatsapp_workflows.sh

echo "Verifying WhatsApp integration..."
./scripts/test_whatsapp_integration.sh
```

## Next Steps

1. **Complete Phase 3**: Ensure production deployment is stable
2. **Start with WhatsApp API**: Build foundation for messaging
3. **Add Voice Processing**: Enable voice note handling
4. **Integrate OpenClaw**: Set up agent orchestration
5. **Extend Workflows**: Add WhatsApp-specific business logic
6. **Implement Cross-Channel**: Ensure consistency across platforms

## Dependencies on Previous Phases

- **Phase 1 Foundation**: Stable core functionality and VAD
- **Phase 2 Intelligence**: RAG, intent classification, workflows
- **Phase 3 Deployment**: Containerization, security, multi-tenancy
- **Infrastructure**: Docker, HTTPS, monitoring systems
- **Configuration**: Environment variables, persona systems

This phase transforms Filo into a truly multi-channel business assistant, enabling Filipino SMEs to serve customers through both web and WhatsApp interfaces while maintaining consistent quality and functionality across all channels.