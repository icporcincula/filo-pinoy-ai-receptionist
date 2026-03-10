# Phase 2 — Intelligence Implementation Plan

## Overview

Phase 2 focuses on adding intelligence to Filo through RAG, intent classification, and workflow automation. This phase builds on the stable foundation established in Phase 1 and adds cognitive capabilities to make Filo more useful for business operations.

## Current State (Post-Phase 1)

Filo has a stable foundation with:
- ✅ Echo cancellation and mic muting
- ✅ WebSocket keep-alive mechanisms
- ✅ Optimized VAD parameters
- ✅ Hot Whisper model loading
- ✅ Local-first privacy architecture

## Phase 2 Goals

Transform Filo from a basic voice receptionist into an intelligent business assistant with:
- **Knowledge Integration**: Business-specific information access
- **Intent Understanding**: Smart categorization of user requests
- **Workflow Automation**: Seamless integration with business processes
- **Enhanced Personalization**: Configurable business personas

## Implementation Components

### 1. Qdrant RAG (Retrieval-Augmented Generation)

**Purpose**: Enable Filo to access business-specific knowledge (FAQs, pricing, hours, policies)

**Implementation**:
```yaml
# docker-compose.yml additions
qdrant:
  image: qdrant/qdrant:latest
  ports:
    - "6333:6333"
  volumes:
    - qdrant_data:/qdrant/storage
  environment:
    - QDRANT__SERVICE__API_KEY=${QDRANT_API_KEY}
```

**Integration Points**:
- Modify LLM pipeline in `server/main.go` to include RAG context
- Add document ingestion system for business docs
- Create RAG query interface for business knowledge

**Files to Create/Modify**:
- `server/rag.go` - RAG integration logic
- `scripts/ingest_docs.py` - Document processing script
- Update `server/main.go` LLM pipeline

### 2. Intent Classification

**Purpose**: Categorize user requests into business-relevant intents (book, inquire, transfer)

**Implementation**:
```go
type Intent string

const (
    IntentBook    Intent = "book"
    IntentInquire Intent = "inquire" 
    IntentTransfer Intent = "transfer"
    IntentUnknown Intent = "unknown"
)

type IntentClassifier struct {
    model *onnx.Model
    intents []Intent
}
```

**Integration Points**:
- Add intent classification before LLM processing
- Route different intents to specialized handling
- Enable intent-based response templates

**Files to Create**:
- `server/intent_classifier.go` - Intent classification logic
- `models/intent_model.onnx` - Pre-trained intent model

### 3. n8n Workflow Integration

**Purpose**: Connect Filo to business workflows and external systems

**Implementation**:
```yaml
# n8n webhook integration
n8n:
  image: n8nio/n8n
  ports:
    - "5678:5678"
  environment:
    - N8N_BASIC_AUTH_ACTIVE=true
    - N8N_BASIC_AUTH_USER=${N8N_USER}
    - N8N_BASIC_AUTH_PASSWORD=${N8N_PASSWORD}
```

**Integration Points**:
- Add webhook handlers for appointment booking
- Create workflow triggers for different intents
- Integrate with Google Calendar API

**Files to Create/Modify**:
- `server/workflow.go` - n8n integration
- `server/calendar.go` - Google Calendar integration
- Update `server/main.go` for workflow routing

### 4. Enhanced Persona Configuration

**Purpose**: Allow businesses to customize Filo's personality and knowledge

**Implementation**:
```env
# .env additions
BUSINESS_NAME=Your Business Name
BUSINESS_HOURS="9:00-18:00"
BUSINESS_LANGUAGE=filipino
PERSONALITY_FRIENDLY=true
PERSONALITY_FORMAL=false
```

**Integration Points**:
- Dynamic system prompt generation
- Business-specific response templates
- Multi-language support

**Files to Create/Modify**:
- `server/persona.go` - Persona management
- `templates/responses/` - Response templates
- Update `.env.example` with new options

## Implementation Timeline

### Week 1: RAG Foundation
- [ ] Set up Qdrant database
- [ ] Create document ingestion pipeline
- [ ] Integrate RAG into LLM pipeline
- [ ] Test knowledge retrieval accuracy

### Week 2: Intent Classification
- [ ] Implement intent classifier
- [ ] Train/test intent detection model
- [ ] Integrate with conversation flow
- [ ] Create intent-specific handlers

### Week 3: Workflow Automation
- [ ] Set up n8n integration
- [ ] Create appointment booking workflow
- [ ] Integrate Google Calendar
- [ ] Test end-to-end workflows

### Week 4: Persona System
- [ ] Implement dynamic persona system
- [ ] Create response template system
- [ ] Add multi-language support
- [ ] Test personalized interactions

## Technical Architecture

### Enhanced Pipeline Flow
```
Browser mic (16kHz PCM)
  │ WebSocket /ws
  ▼
Go server :8080
  ├── TEN VAD        → ONNX-based speech detection
  ├── Intent Classifier → Categorize user intent
  ├── RAG System     → Retrieve business knowledge
  ├── STT   →   Speaches :8000
  ├── LLM   →   Ollama   :11434 (with RAG context)
  ├── Workflow Router → Route to n8n workflows
  ├── TTS   →   Kokoro   :5000
  └── History → Redis    :6379
```

### Data Flow
1. **Voice Input** → VAD detection
2. **STT** → Text transcription
3. **Intent Classification** → Determine request type
4. **RAG Retrieval** → Fetch relevant business knowledge
5. **LLM Processing** → Generate response with context
6. **Workflow Routing** → Trigger appropriate workflows
7. **TTS Output** → Voice response

## Dependencies and Requirements

### New Dependencies
- **Qdrant**: Vector database for RAG
- **ONNX Runtime**: For intent classification model
- **n8n**: Workflow automation platform
- **Google Calendar API**: For appointment management

### Configuration Requirements
- Business document storage location
- n8n workflow definitions
- Intent classification model training data
- Google Calendar API credentials

## Success Criteria

### Functional Requirements
- [ ] RAG system retrieves relevant business information >90% accuracy
- [ ] Intent classification achieves >85% accuracy
- [ ] Appointment booking workflow completes successfully >95% of the time
- [ ] Response personalization improves user satisfaction

### Performance Requirements
- [ ] RAG retrieval adds <500ms latency
- [ ] Intent classification adds <100ms latency
- [ ] Workflow integration maintains real-time responsiveness
- [ ] System handles concurrent business operations

### Business Requirements
- [ ] Support for common business intents (booking, inquiries, transfers)
- [ ] Integration with standard business tools (Google Calendar, email)
- [ ] Configurable business personas and knowledge bases
- [ ] Multi-language support for Filipino businesses

## Risk Mitigation

### Technical Risks
- **RAG Performance**: Monitor and optimize retrieval latency
- **Intent Accuracy**: Implement fallback mechanisms for unknown intents
- **Workflow Reliability**: Add error handling and retry mechanisms
- **Data Privacy**: Ensure business data remains local and secure

### Integration Risks
- **API Dependencies**: Handle external API failures gracefully
- **Model Updates**: Plan for intent model retraining and updates
- **Business Logic**: Allow for business-specific workflow customization

## Next Steps

1. **Complete Phase 1**: Ensure stable foundation
2. **Start with RAG**: Build knowledge foundation first
3. **Add Intent Classification**: Enable smart routing
4. **Integrate Workflows**: Connect to business processes
5. **Enhance Personas**: Add business-specific customization

## Dependencies on Previous Phases

- **Phase 1 Foundation**: Stable VAD, WebSocket, and LLM pipeline
- **Infrastructure**: Docker setup, Redis, and service orchestration
- **Configuration System**: Environment variable management
- **Monitoring**: Logging and error handling infrastructure

This phase transforms Filo from a basic voice interface into an intelligent business assistant capable of handling complex business operations while maintaining the local-first privacy principles.