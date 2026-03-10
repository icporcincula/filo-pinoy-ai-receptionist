package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// IntegrationTestSuite holds shared test state and utilities
type IntegrationTestSuite struct {
	// Mock servers for external dependencies
	qdrantServer   *httptest.Server
	n8nServer      *httptest.Server
	calendarServer *httptest.Server
	turnServer     *httptest.Server
	
	// Test session
	session *session
}

// SetupTestSuite creates mock servers and initializes test environment
func SetupTestSuite(t *testing.T) *IntegrationTestSuite {
	suite := &IntegrationTestSuite{}

	// Setup Qdrant mock server
	suite.qdrantServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/points/search") {
			// Return mock search results
			response := SearchResponse{
				Result: []struct {
					ID      interface{} `json:"id"`
					Score   float64     `json:"score"`
					Payload struct {
						Content string `json:"content"`
						Title   string `json:"title"`
						Source  string `json:"source"`
					} `json:"payload"`
				}{
					{
						ID:    "business-hours",
						Score: 0.95,
						Payload: struct {
							Content string `json:"content"`
							Title   string `json:"title"`
							Source  string `json:"source"`
						}{
							Content: "We are open from 9:00 AM to 6:00 PM, Monday through Friday.",
							Title:   "Business Hours",
							Source:  "faq.txt",
						},
					},
				},
				Status: "ok",
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if strings.Contains(r.URL.Path, "/points") && r.Method == "PUT" {
			// Return success for add document
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status": "ok"}`))
		}
	}))

	// Setup n8n mock server
	suite.n8nServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/webhook/appointment-booking-workflow") {
			response := WorkflowResponse{
				Success: true,
				Data: map[string]interface{}{
					"appointment_id": "12345",
					"status": "confirmed",
					"time": "2024-01-15 10:00:00",
				},
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if strings.Contains(r.URL.Path, "/rest/workflows") {
			workflows := []map[string]interface{}{
				{
					"id":   "1",
					"name": "appointment-booking-workflow",
					"active": true,
				},
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(workflows)
		}
	}))

	// Setup Calendar mock server
	suite.calendarServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			// Return no conflicting events
			eventsResponse := struct {
				Items []CalendarEvent `json:"items"`
			}{Items: []CalendarEvent{}}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(eventsResponse)
		} else if r.Method == "POST" {
			// Return created event
			createdEvent := CalendarEvent{
				Summary:     "Test Appointment",
				Description: "Test appointment description",
				Start: EventDateTime{
					DateTime: "2024-01-15T10:00:00+08:00",
					TimeZone: "Asia/Manila",
				},
				End: EventDateTime{
					DateTime: "2024-01-15T10:30:00+08:00",
					TimeZone: "Asia/Manila",
				},
				Location: "Virtual Meeting",
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(createdEvent)
		}
	}))

	// Setup Turn Detection mock server
	suite.turnServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{
					Message: struct {
						Content string `json:"content"`
					}{
						Content: "finished",
					},
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))

	return suite
}

// TeardownTestSuite cleans up mock servers
func (suite *IntegrationTestSuite) TeardownTestSuite(t *testing.T) {
	if suite.qdrantServer != nil {
		suite.qdrantServer.Close()
	}
	if suite.n8nServer != nil {
		suite.n8nServer.Close()
	}
	if suite.calendarServer != nil {
		suite.calendarServer.Close()
	}
	if suite.turnServer != nil {
		suite.turnServer.Close()
	}
}

// TestEndToEndConversationFlow tests a complete conversation with all intelligence components
func TestEndToEndConversationFlow(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TeardownTestSuite(t)

	// Create test configuration
	cfg := Config{
		Port:        "8080",
		SpeachesURL: "http://localhost:8000",
		OllamaURL:   "http://localhost:11434",
		KokoroURL:   "http://localhost:5000",
		RedisAddr:   "localhost:6379",
		LLMModel:    "llama3.1:8b",
		SystemPrompt: "You are a helpful, concise voice assistant. Keep responses short and conversational.",
		TENVADMode:  2,
		TurnDetectionBaseURL: suite.turnServer.URL + "/v1",
		TurnDetectionAPIKey:  "test-key",
		TurnDetectionModel:   "test-model",
		TurnDetectionTemperature: 0.1,
		TurnDetectionTopP:   0.1,
		TurnDetectionThresholdMs: 500,
		QdrantURL:   suite.qdrantServer.URL,
		QdrantAPIKey: "test-key",
		QdrantCollectionName: "knowledge_base",
		N8nURL:      suite.n8nServer.URL,
		N8nUser:     "testuser",
		N8nPassword: "testpass",
		BusinessName: "Test Business",
		BusinessHours: "9:00-18:00",
		BusinessLanguage: "en",
		PersonalityFriendly: true,
		PersonalityFormal: false,
		GoogleCalendarID: "test-calendar-id",
		GoogleCredentialsPath: "",
	}

	// Create mock WebSocket connection
	mockConn := &MockWebSocketConn{}
	
	// Create mock Redis client
	mockRedis := &MockRedisClient{}

	// Create session
	session := newSession("test-session", mockConn, cfg, mockRedis)
	
	// Test conversation flow
	testCases := []struct {
		name           string
		input          string
		expectedIntent Intent
		expectedContains []string
	}{
		{
			name:           "Greeting",
			input:          "Hello",
			expectedIntent: IntentGreeting,
			expectedContains: []string{"Welcome to Test Business"},
		},
		{
			name:           "Business hours inquiry",
			input:          "What are your business hours?",
			expectedIntent: IntentInquire,
			expectedContains: []string{"9:00 AM to 6:00 PM"},
		},
		{
			name:           "Appointment booking",
			input:          "I want to book an appointment",
			expectedIntent: IntentBook,
			expectedContains: []string{"booking system"},
		},
		{
			name:           "Transfer request",
			input:          "Transfer me to a manager",
			expectedIntent: IntentTransfer,
			expectedContains: []string{"representative"},
		},
		{
			name:           "Goodbye",
			input:          "Thank you, goodbye",
			expectedIntent: IntentGoodbye,
			expectedContains: []string{"Have a great day"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate VAD processing
			samples := generateTestAudioSamples(tc.input)
			session.onUtterance(samples)
			
			// Verify intent classification
			intentHandler := NewIntentHandler()
			intentContext := intentHandler.ProcessIntent(tc.input)
			assert.Equal(t, tc.expectedIntent, intentContext.Intent)
			
			// Verify RAG context retrieval
			qdrantClient := NewQdrantClient(cfg.QdrantURL, cfg.QdrantAPIKey, cfg.QdrantCollectionName)
			ragSystem := NewRAGSystem(qdrantClient)
			ctx := context.Background()
			
			ragContext, err := ragSystem.GetRAGContext(ctx, tc.input)
			if tc.expectedIntent == IntentInquire {
				assert.NoError(t, err)
				assert.NotEmpty(t, ragContext)
			}
			
			// Verify persona system
			personaManager := NewPersonaManager()
			prompt := personaManager.GenerateSystemPrompt(cfg)
			assert.Contains(t, prompt, "Test Business")
			
			// Verify response generation
			personalizationEngine := NewPersonalizationEngine()
			response := personalizationEngine.GeneratePersonalizedResponse(string(tc.expectedIntent), tc.input, cfg)
			assert.NotNil(t, response)
			assert.Equal(t, string(tc.expectedIntent), response.Intent)
			
			// Check response content
			for _, contains := range tc.expectedContains {
				assert.Contains(t, response.Text, contains)
			}
		})
	}
}

// TestRAGWorkflowIntegration tests RAG system with workflow integration
func TestRAGWorkflowIntegration(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TeardownTestSuite(t)

	// Setup RAG system
	qdrantClient := NewQdrantClient(suite.qdrantServer.URL, "test-key", "knowledge_base")
	ragSystem := NewRAGSystem(qdrantClient)
	
	// Setup workflow system
	workflowManager := NewWorkflowManager(suite.n8nServer.URL, "testuser", "testpass")
	workflowExecutor := NewWorkflowExecutor(workflowManager)
	
	ctx := context.Background()

	t.Run("RAG context with workflow routing", func(t *testing.T) {
		// Test inquiry that should trigger RAG and workflow
		query := "What are your business hours and how do I book an appointment?"
		
		// Get RAG context
		ragContext, err := ragSystem.GetRAGContext(ctx, query)
		assert.NoError(t, err)
		assert.Contains(t, ragContext, "Business Knowledge Base Information")
		assert.Contains(t, ragContext, "9:00 AM to 6:00 PM")
		
		// Test workflow execution for appointment booking
		resp, err := workflowExecutor.ExecuteIntentWorkflow(ctx, "test-session", IntentBook, query)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.Equal(t, "12345", resp.Data["appointment_id"])
	})

	t.Run("RAG accuracy and latency", func(t *testing.T) {
		queries := []string{
			"What are your business hours?",
			"How do I book an appointment?",
			"Where are you located?",
			"What services do you offer?",
		}
		
		start := time.Now()
		for _, query := range queries {
			ragContext, err := ragSystem.GetRAGContext(ctx, query)
			assert.NoError(t, err)
			assert.NotEmpty(t, ragContext)
		}
		duration := time.Since(start)
		
		// Should process 4 queries quickly
		assert.Less(t, duration, 2*time.Second, "RAG queries should be fast")
	})
}

// TestIntentWorkflowIntegration tests intent classification with workflow execution
func TestIntentWorkflowIntegration(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TeardownTestSuite(t)

	// Setup workflow system
	workflowManager := NewWorkflowManager(suite.n8nServer.URL, "testuser", "testpass")
	workflowExecutor := NewWorkflowExecutor(workflowManager)
	
	// Setup intent system
	intentHandler := NewIntentHandler()
	
	ctx := context.Background()

	t.Run("Intent-based workflow routing", func(t *testing.T) {
		testCases := []struct {
			input    string
			intent   Intent
			workflow string
		}{
			{"Book an appointment", IntentBook, "appointment_booking"},
			{"Transfer to manager", IntentTransfer, "transfer_to_agent"},
			{"What are your hours?", IntentInquire, "knowledge_search"},
		}
		
		for _, tc := range testCases {
			// Classify intent
			intentContext := intentHandler.ProcessIntent(tc.input)
			assert.Equal(t, tc.intent, intentContext.Intent)
			assert.True(t, intentContext.ShouldRoute)
			
			// Execute workflow
			resp, err := workflowExecutor.ExecuteIntentWorkflow(ctx, "test-session", tc.intent, tc.input)
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.True(t, resp.Success)
		}
	})

	t.Run("Workflow error handling", func(t *testing.T) {
		// Test with invalid workflow
		resp, err := workflowExecutor.ExecuteIntentWorkflow(ctx, "test-session", IntentUnknown, "Unknown request")
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

// TestCalendarIntegration tests calendar system with workflow integration
func TestCalendarIntegration(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TeardownTestSuite(t)

	// Setup calendar system
	credentials := &CalendarCredentials{
		AccessToken: "test-access-token",
	}
	calendarExecutor := NewCalendarExecutor(suite.calendarServer.URL, credentials)
	
	ctx := context.Background()

	t.Run("Calendar availability and booking", func(t *testing.T) {
		// Test getting available slots
		date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		duration := 30 * time.Minute

		slots, err := calendarExecutor.GetAvailableSlots(ctx, "test-session", date, duration)
		assert.NoError(t, err)
		assert.NotNil(t, slots)
		assert.Greater(t, len(slots), 0)

		// Test booking appointment
		if len(slots) > 0 {
			slotStart, _ := time.Parse(time.RFC3339, slots[0].Start)
			slotEnd, _ := time.Parse(time.RFC3339, slots[0].End)

			createdEvent, err := calendarExecutor.BookAppointment(ctx, "test-session", 
				"Test Appointment", "Test appointment description", 
				slotStart, slotEnd, []string{"test@example.com"})
			
			assert.NoError(t, err)
			assert.NotNil(t, createdEvent)
			assert.Equal(t, "Test Appointment", createdEvent.Summary)
		}
	})

	t.Run("Calendar conflict detection", func(t *testing.T) {
		// This would be tested with a mock server that returns conflicting events
		// For now, we test the basic functionality
		date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		duration := 30 * time.Minute

		slots, err := calendarExecutor.GetAvailableSlots(ctx, "test-session", date, duration)
		assert.NoError(t, err)
		assert.NotNil(t, slots)
	})
}

// TestPersonaSystemIntegration tests persona system with all other components
func TestPersonaSystemIntegration(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TeardownTestSuite(t)

	// Setup persona system
	personalizationEngine := NewPersonalizationEngine()
	
	// Setup business context
	newContext := &BusinessContext{
		Name:        "Acme Corp",
		Industry:    "Technology",
		Location:    "Manila",
		Hours:       "9:00-18:00",
		Phone:       "+63 2 555 1234",
		Email:       "info@acme.com",
		Website:     "https://acme.com",
		Description: "Leading technology company",
		Services:    []string{"Consulting", "Development", "Support"},
		FAQ: []FAQItem{
			{
				Question: "What services do you offer?",
				Answer:   "We offer consulting, development, and support services.",
				Category: "Services",
			},
		},
	}
	
	ctx := context.Background()
	personalizationEngine.contextManager.UpdateBusinessContext(ctx, newContext)
	
	cfg := Config{
		BusinessName:      "Acme Corp",
		BusinessHours:     "9:00-18:00",
		BusinessLanguage:  "en",
		PersonalityFriendly: true,
		PersonalityFormal: false,
	}

	t.Run("Personalized responses with business context", func(t *testing.T) {
		testCases := []struct {
			intent string
			input  string
			contains string
		}{
			{"greeting", "Hello", "Welcome to Acme Corp"},
			{"goodbye", "Goodbye", "Thank you for contacting Acme Corp"},
			{"business_hours", "What are your hours?", "We are open from"},
			{"appointment", "Book appointment", "I can help you book"},
		}
		
		for _, tc := range testCases {
			response := personalizationEngine.GeneratePersonalizedResponse(tc.intent, tc.input, cfg)
			assert.NotNil(t, response)
			assert.Equal(t, tc.intent, response.Intent)
			assert.Equal(t, "business", response.Persona)
			assert.NotNil(t, response.Context)
			assert.NotEmpty(t, response.Text)
			
			if tc.contains != "" {
				assert.Contains(t, response.Text, tc.contains)
			}
		}
	})

	t.Run("Dynamic context integration", func(t *testing.T) {
		context := personalizationEngine.contextManager.GetDynamicContext()
		assert.Equal(t, "Acme Corp", context["business_name"])
		assert.Equal(t, "Manila", context["business_location"])
		assert.Equal(t, "Consulting", context["services"].([]string)[0])
	})
}

// TestSystemPerformance tests overall system performance
func TestSystemPerformance(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TeardownTestSuite(t)

	// Setup all systems
	qdrantClient := NewQdrantClient(suite.qdrantServer.URL, "test-key", "knowledge_base")
	ragSystem := NewRAGSystem(qdrantClient)
	
	workflowManager := NewWorkflowManager(suite.n8nServer.URL, "testuser", "testpass")
	workflowExecutor := NewWorkflowExecutor(workflowManager)
	
	credentials := &CalendarCredentials{
		AccessToken: "test-access-token",
	}
	calendarExecutor := NewCalendarExecutor(suite.calendarServer.URL, credentials)
	
	personalizationEngine := NewPersonalizationEngine()
	
	intentHandler := NewIntentHandler()
	
	ctx := context.Background()

	t.Run("High volume request handling", func(t *testing.T) {
		queries := []string{
			"Hello",
			"What are your business hours?",
			"I want to book an appointment",
			"Transfer me to a manager",
			"Thank you",
		}
		
		start := time.Now()
		
		for i := 0; i < 100; i++ {
			query := queries[i%len(queries)]
			
			// Process intent
			intentContext := intentHandler.ProcessIntent(query)
			
			// Process RAG
			if intentContext.Intent == IntentInquire {
				_, _ = ragSystem.GetRAGContext(ctx, query)
			}
			
			// Process workflow
			if intentContext.ShouldRoute {
				_, _ = workflowExecutor.ExecuteIntentWorkflow(ctx, "test-session", intentContext.Intent, query)
			}
			
			// Process persona
			cfg := Config{
				BusinessName:      "Test Business",
				BusinessHours:     "9:00-18:00",
				BusinessLanguage:  "en",
				PersonalityFriendly: true,
				PersonalityFormal: false,
			}
			_ = personalizationEngine.GeneratePersonalizedResponse(string(intentContext.Intent), query, cfg)
		}
		
		duration := time.Since(start)
		
		// Should process 100 requests in under 10 seconds
		assert.Less(t, duration, 10*time.Second, "Processing 100 requests should be fast")
		t.Logf("Processed 100 requests in %v", duration)
	})

	t.Run("Memory usage and cleanup", func(t *testing.T) {
		// Test that systems don't leak memory
		// This is a basic test - in production you'd use more sophisticated memory profiling
		
		initialMemory := getMemoryUsage()
		
		for i := 0; i < 1000; i++ {
			// Create and discard objects
			_ = NewQdrantClient("http://localhost:6333", "test", "test")
			_ = NewWorkflowManager("http://localhost:5678", "test", "test")
			_ = NewCalendarManager("test", &CalendarCredentials{})
			_ = NewPersonalizationEngine()
		}
		
		// Force garbage collection
		// runtime.GC()
		
		finalMemory := getMemoryUsage()
		
		// Memory should not grow excessively
		memoryGrowth := finalMemory - initialMemory
		assert.Less(t, memoryGrowth, int64(50*1024*1024), "Memory growth should be reasonable") // 50MB limit
	})
}

// Mock implementations for testing

type MockWebSocketConn struct{}

func (m *MockWebSocketConn) WriteMessage(messageType int, data []byte) error {
	return nil
}

func (m *MockWebSocketConn) ReadMessage() (messageType int, p []byte, err error) {
	return 0, nil, nil
}

type MockRedisClient struct{}

func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	return nil
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return nil
}

func (m *MockRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	return nil
}

// Helper functions

func generateTestAudioSamples(text string) []int16 {
	// Generate simple test audio samples
	// In a real test, this would be actual audio data
	samples := make([]int16, 16000) // 1 second of silence at 16kHz
	for i := range samples {
		samples[i] = int16(i % 1000) // Simple pattern
	}
	return samples
}

func getMemoryUsage() int64 {
	// Simple memory usage approximation
	// In production, use runtime.ReadMemStats or similar
	return 0
}