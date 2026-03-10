# TEN VAD and Turn Detection Implementation Plan

## Overview

This document outlines the plan to integrate TEN Framework's superior VAD (Voice Activity Detection) and Turn Detection capabilities into Filo, replacing the current energy-based VAD with ONNX-based TEN VAD and adding intelligent turn detection for better conversation flow.

## Current State Analysis

### Filo's Current Implementation
- **VAD**: Simple energy-based detection in `server/main.go`
  - Fixed thresholds: `vadEnergyThresh = 100`, `vadSilenceMs = 1200`
  - Only detects silence vs speech, no semantic understanding
  - No turn detection - waits for silence then processes
- **Repository Size**: ~10 core files + 1.2GB TEN framework directory

### TEN Framework Components Available
1. **TEN VAD** (`ten-framework/packages/example_extensions/webrtc_vad_cpp/`)
   - ONNX-based, outperforms WebRTC and Silero VAD
   - Lower computational complexity and memory usage
   - No PyTorch dependency required
2. **TEN Turn Detection** (`ten-framework/ai_agents/agents/ten_packages/extension/ten_turn_detection/`)
   - AI-powered semantic turn detection
   - Categorizes speech into: finished/unfinished/unknown states
   - Prevents mid-sentence interruptions

## Implementation Strategy

### Phase 1: Extract TEN Components (Priority: HIGH)
**Goal**: Extract and adapt TEN VAD and Turn Detection components before removing the framework

#### 1.1 Extract TEN VAD Components
**Source**: `ten-framework/packages/example_extensions/webrtc_vad_cpp/`
**Target**: Convert to Go implementation

**Files to Extract**:
- `third_party/webrtc_vad/webrtc_vad.h` - VAD API definitions
- `third_party/webrtc_vad/webrtc_vad.c` - Core VAD implementation
- `src/main.cc` - Extension logic and integration patterns

**Adaptation Requirements**:
- Convert C++ VAD extension to Go
- Maintain same interface: `feed(samples []int16)` and callback system
- Integrate ONNX model loading and inference
- Handle 16kHz mono PCM input (same as current)

#### 1.2 Extract TEN Turn Detection Components
**Source**: `ten-framework/ai_agents/agents/ten_packages/extension/ten_turn_detection/`
**Target**: Convert to Go implementation

**Files to Extract**:
- `turn_detector.py` - Core turn detection logic
- `extension.py` - Extension interface and state management
- `config.py` - Configuration handling

**Adaptation Requirements**:
- Convert Python async logic to Go goroutines
- Implement semantic analysis for speech states
- Create API client for turn detection model
- Handle three states: finished/unfinished/unknown

### Phase 2: Integrate with Filo Pipeline
**Goal**: Replace current VAD and add turn detection to existing pipeline

#### 2.1 Update VAD Implementation
**File**: `server/main.go`
**Changes**:
- Replace `vad` struct with TEN VAD implementation
- Update VAD constants to use TEN's optimized values
- Maintain existing callback interface for backward compatibility

#### 2.2 Add Turn Detection Logic
**File**: `server/main.go`
**Changes**:
- Add `turnDetector` struct
- Modify `onUtterance` to use turn detection before processing
- Implement state machine for turn detection decisions
- Add configuration for turn detection API

#### 2.3 Update Configuration
**File**: `.env.example`
**Changes**:
- Add TEN VAD configuration options
- Add turn detection API settings
- Maintain backward compatibility with existing config

### Phase 3: Remove TEN Framework (Priority: LOW)
**Goal**: Clean up repository after extracting needed components

**Action**: Remove entire `ten-framework/` directory
**Impact**: Reduce repository size by ~1.2GB
**Timing**: After successful integration and testing

## Technical Architecture

### New Pipeline Flow
```
Browser mic (16kHz PCM)
  │ WebSocket /ws
  ▼
Go server :8080
  ├── TEN VAD        → ONNX-based speech detection
  ├── Turn Detection → Semantic analysis (finished/unfinished/unknown)
  ├── STT   →   Speaches :800
  ├── LLM   →   Ollama   :11434
  ├── TTS   →   Kokoro   :5000
  └── History → Redis    :6379
```

### Key Components

#### TEN VAD Go Implementation
```go
type tenVAD struct {
    model     *onnx.Model
    threshold float64
    onUtterance func([]int16)
}

func (v *tenVAD) feed(samples []int16) {
    // ONNX inference for speech detection
    // Maintain existing callback interface
}
```

#### Turn Detection Go Implementation
```go
type turnDetector struct {
    apiClient *http.Client
    config    TurnDetectionConfig
}

type TurnState int

const (
    TurnStateUnfinished TurnState = iota
    TurnStateFinished
    TurnStateUnknown
)

func (td *turnDetector) evaluate(text string) (TurnState, error) {
    // Send text to turn detection API
    // Return semantic analysis result
}
```

## Implementation Timeline

### Week 1: Extraction and Conversion
- [ ] Extract TEN VAD C++ implementation
- [ ] Convert to Go with ONNX integration
- [ ] Extract TEN Turn Detection Python implementation
- [ ] Convert to Go with HTTP client

### Week 2: Integration and Testing
- [ ] Integrate TEN VAD into Filo pipeline
- [ ] Add turn detection logic
- [ ] Update configuration system
- [ ] Basic functionality testing

### Week 3: Optimization and Validation
- [ ] Performance optimization
- [ ] Accuracy validation vs current VAD
- [ ] Turn detection behavior testing
- [ ] Integration testing

### Week 4: Cleanup and Documentation
- [ ] Remove TEN framework directory
- [ ] Update documentation
- [ ] Final testing and validation
- [ ] Prepare release notes

## Benefits

### Improved VAD Performance
- **Higher accuracy**: ONNX-based vs energy threshold
- **Lower latency**: Optimized computational complexity
- **Better noise handling**: Advanced speech detection algorithms

### Intelligent Turn Taking
- **No interruptions**: Semantic understanding prevents mid-sentence cuts
- **Faster responses**: Quick detection of finished thoughts
- **Natural conversation**: Human-like turn-taking behavior

### Repository Optimization
- **Smaller size**: Remove 1.2GB of unused framework code
- **Cleaner codebase**: Focused implementation without framework overhead
- **Easier maintenance**: Simplified dependency management

## Risk Mitigation

### Technical Risks
- **ONNX dependency**: Ensure proper Go ONNX library integration
- **API availability**: Handle turn detection API failures gracefully
- **Performance impact**: Monitor and optimize inference latency

### Migration Risks
- **Backward compatibility**: Maintain existing VAD interface
- **Configuration changes**: Provide migration path for existing setups
- **Testing coverage**: Comprehensive testing of new components

## Success Criteria

### Functional Requirements
- [ ] TEN VAD accuracy > current energy-based VAD
- [ ] Turn detection reduces interruptions by >50%
- [ ] End-to-end latency improvement >20%
- [ ] 100% backward compatibility with existing Filo functionality

### Non-Functional Requirements
- [ ] Repository size reduction >1GB
- [ ] No new external dependencies beyond ONNX
- [ ] Maintain current performance benchmarks
- [ ] Zero breaking changes to existing API

## Next Steps

1. **Toggle to Act Mode** to begin implementation
2. Start with Phase 1: Extract TEN VAD components
3. Convert to Go implementation with ONNX integration
4. Proceed through integration phases systematically
5. Remove TEN framework only after successful validation

## References

- [TEN Framework Repository](https://github.com/TEN-framework/ten-framework)
- [TEN VAD Implementation](ten-framework/packages/example_extensions/webrtc_vad_cpp/)
- [TEN Turn Detection Implementation](ten-framework/ai_agents/agents/ten_packages/extension/ten_turn_detection/)
- [Filo Current Implementation](server/main.go)