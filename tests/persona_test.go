package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test Persona struct and basic functionality
func TestPersona(t *testing.T) {
	t.Run("NewPersonaManager", func(t *testing.T) {
		manager := NewPersonaManager()
		assert.NotNil(t, manager)
		assert.NotNil(t, manager.personas)
		assert.NotNil(t, manager.defaultPersona)
		assert.Equal(t, "Default", manager.defaultPersona.Name)
		assert.Equal(t, "professional", manager.defaultPersona.Tone)
		assert.Equal(t, "en", manager.defaultPersona.Language)
		assert.Equal(t, "neutral", manager.defaultPersona.Formality)
	})

	t.Run("AddPersona", func(t *testing.T) {
		manager := NewPersonaManager()
		
		persona := &Persona{
			Name:        "Business",
			Description: "Business assistant persona",
			Tone:        "professional",
			Language:    "en",
			Formality:   "formal",
			Keywords:    []string{"help", "business"},
			Responses: map[string]string{
				"greeting": "Welcome to our business",
			},
		}
		
		manager.AddPersona(persona)
		retrieved := manager.GetPersona("Business")
		assert.NotNil(t, retrieved)
		assert.Equal(t, "Business", retrieved.Name)
		assert.Equal(t, "Business assistant persona", retrieved.Description)
	})

	t.Run("GetPersona", func(t *testing.T) {
		manager := NewPersonaManager()
		
		// Test getting existing persona
		persona := manager.GetPersona("Business")
		assert.NotNil(t, persona)
		assert.Equal(t, "Default", persona.Name)
		
		// Test getting non-existing persona returns default
		manager.AddPersona(&Persona{Name: "Test"})
		persona = manager.GetPersona("NonExistent")
		assert.NotNil(t, persona)
		assert.Equal(t, "Default", persona.Name)
	})

	t.Run("GetDefaultPersona", func(t *testing.T) {
		manager := NewPersonaManager()
		defaultPersona := manager.GetDefaultPersona()
		assert.NotNil(t, defaultPersona)
		assert.Equal(t, "Default", defaultPersona.Name)
	})
}

// Test GenerateSystemPrompt
func TestGenerateSystemPrompt(t *testing.T) {
	t.Run("GenerateSystemPrompt with friendly formal", func(t *testing.T) {
		manager := NewPersonaManager()
		cfg := Config{
			BusinessName:      "Test Business",
			BusinessHours:     "9:00-18:00",
			BusinessLanguage:  "en",
			PersonalityFriendly: true,
			PersonalityFormal: true,
		}
		
		prompt := manager.GenerateSystemPrompt(cfg)
		assert.Contains(t, prompt, "Test Business")
		assert.Contains(t, prompt, "9:00-18:00")
		assert.Contains(t, prompt, "formal and professional")
		assert.Contains(t, prompt, "helpful, friendly, and personable")
	})

	t.Run("GenerateSystemPrompt with unfriendly informal", func(t *testing.T) {
		manager := NewPersonaManager()
		cfg := Config{
			BusinessName:      "Test Business",
			BusinessHours:     "9:00-18:00",
			BusinessLanguage:  "en",
			PersonalityFriendly: false,
			PersonalityFormal: false,
		}
		
		prompt := manager.GenerateSystemPrompt(cfg)
		assert.Contains(t, prompt, "Test Business")
		assert.Contains(t, prompt, "9:00-18:00")
		assert.Contains(t, prompt, "friendly and conversational")
		assert.Contains(t, prompt, "professional and to the point")
	})

	t.Run("GenerateSystemPrompt with different language", func(t *testing.T) {
		manager := NewPersonaManager()
		cfg := Config{
			BusinessName:      "Test Business",
			BusinessHours:     "9:00-18:00",
			BusinessLanguage:  "fil",
			PersonalityFriendly: true,
			PersonalityFormal: false,
		}
		
		prompt := manager.GenerateSystemPrompt(cfg)
		assert.Contains(t, prompt, "Test Business")
		assert.Contains(t, prompt, "9:00-18:00")
		assert.Contains(t, prompt, "Respond in fil")
	})
}

// Test ContextManager
func TestContextManager(t *testing.T) {
	t.Run("NewContextManager", func(t *testing.T) {
		manager := NewContextManager()
		assert.NotNil(t, manager)
		assert.NotNil(t, manager.businessContext)
		assert.NotNil(t, manager.personaManager)
		assert.Equal(t, "Your Business Name", manager.businessContext.Name)
		assert.Equal(t, "Service", manager.businessContext.Industry)
		assert.Equal(t, "Philippines", manager.businessContext.Location)
	})

	t.Run("UpdateBusinessContext", func(t *testing.T) {
		manager := NewContextManager()
		
		newContext := &BusinessContext{
			Name:        "Updated Business",
			Industry:    "Technology",
			Location:    "Manila",
			Hours:       "8:00-17:00",
			Phone:       "+63 2 123 4567",
			Email:       "info@updated.com",
			Website:     "https://updated.com",
			Description: "Updated business description",
			Services:    []string{"Tech", "Support"},
			FAQ: []FAQItem{
				{
					Question: "Updated question?",
					Answer:   "Updated answer",
					Category: "Updated",
				},
			},
		}
		
		ctx := context.Background()
		manager.UpdateBusinessContext(ctx, newContext)
		
		retrieved := manager.GetBusinessContext()
		assert.NotNil(t, retrieved)
		assert.Equal(t, "Updated Business", retrieved.Name)
		assert.Equal(t, "Technology", retrieved.Industry)
		assert.Equal(t, "Manila", retrieved.Location)
		assert.Equal(t, "8:00-17:00", retrieved.Hours)
		assert.Equal(t, "Tech", retrieved.Services[0])
		assert.Equal(t, "Updated question?", retrieved.FAQ[0].Question)
	})

	t.Run("GetDynamicContext", func(t *testing.T) {
		manager := NewContextManager()
		context := manager.GetDynamicContext()
		
		assert.NotNil(t, context)
		assert.Equal(t, "Your Business Name", context["business_name"])
		assert.Equal(t, "9:00 AM - 6:00 PM", context["business_hours"])
		assert.Equal(t, "Philippines", context["business_location"])
		assert.Equal(t, "Services", context["services"])
		assert.Equal(t, float64(2), context["faq_count"]) // Default has 2 FAQ items
	})
}

// Test TemplateManager
func TestTemplateManager(t *testing.T) {
	t.Run("NewTemplateManager", func(t *testing.T) {
		manager := NewTemplateManager()
		assert.NotNil(t, manager)
		assert.NotNil(t, manager.templates)
		assert.Len(t, manager.templates, 7) // greeting, goodbye, unknown, business_hours, appointment, inquiry
	})

	t.Run("GetTemplate", func(t *testing.T) {
		manager := NewTemplateManager()
		
		// Test getting greeting template
		template := manager.GetTemplate("greeting", map[string]interface{}{
			"time_of_day": "morning",
		})
		assert.Contains(t, template, "Welcome to")
		assert.Contains(t, template, "{{business_name}}")
		
		// Test getting goodbye template
		template = manager.GetTemplate("goodbye", map[string]interface{}{})
		assert.Contains(t, template, "Thank you for contacting")
		assert.Contains(t, template, "{{business_name}}")
		
		// Test getting unknown template
		template = manager.GetTemplate("unknown", map[string]interface{}{})
		assert.Contains(t, template, "rephrase your question")
		
		// Test getting template for non-existent intent
		template = manager.GetTemplate("nonexistent", map[string]interface{}{})
		assert.Empty(t, template)
	})
}

// Test PersonalizationEngine
func TestPersonalizationEngine(t *testing.T) {
	t.Run("NewPersonalizationEngine", func(t *testing.T) {
		engine := NewPersonalizationEngine()
		assert.NotNil(t, engine)
		assert.NotNil(t, engine.contextManager)
		assert.NotNil(t, engine.templateManager)
		assert.NotNil(t, engine.personaManager)
	})

	t.Run("GeneratePersonalizedResponse", func(t *testing.T) {
		engine := NewPersonalizationEngine()
		cfg := Config{
			BusinessName:      "Test Business",
			BusinessHours:     "9:00-18:00",
			BusinessLanguage:  "en",
			PersonalityFriendly: true,
			PersonalityFormal: false,
		}
		
		response := engine.GeneratePersonalizedResponse("greeting", "Hello", cfg)
		assert.NotNil(t, response)
		assert.Equal(t, "greeting", response.Intent)
		assert.Equal(t, "business", response.Persona)
		assert.NotNil(t, response.Context)
		assert.NotEmpty(t, response.Text)
		assert.Contains(t, response.Text, "Test Business")
		
		// Test with unknown intent
		response = engine.GeneratePersonalizedResponse("unknown", "Random text", cfg)
		assert.NotNil(t, response)
		assert.Equal(t, "unknown", response.Intent)
		assert.Equal(t, "business", response.Persona)
		assert.NotNil(t, response.Context)
		assert.NotEmpty(t, response.Text)
	})

	t.Run("fillTemplate", func(t *testing.T) {
		engine := NewPersonalizationEngine()
		
		template := "Hello {{business_name}}! Today is {{day_of_week}}."
		context := map[string]interface{}{
			"business_name": "Test Business",
			"day_of_week":   "Monday",
		}
		
		result := engine.fillTemplate(template, context)
		assert.Equal(t, "Hello Test Business! Today is Monday.", result)
		
		// Test with missing context
		template = "Hello {{business_name}}! Today is {{day_of_week}}."
		context = map[string]interface{}{
			"business_name": "Test Business",
		}
		
		result = engine.fillTemplate(template, context)
		assert.Equal(t, "Hello Test Business! Today is <no value>.", result)
	})
}

// Test PersonaLogger
func TestPersonaLogger(t *testing.T) {
	t.Run("LogPersonaInteraction", func(t *testing.T) {
		logger := &PersonaLogger{}
		response := &PersonalizedResponse{
			Text:      "Test response",
			Intent:    "greeting",
			Persona:   "business",
			Context:   map[string]interface{}{"test": "data"},
			Timestamp: time.Now().Unix(),
		}
		
		// This should not panic and should log the interaction
		logger.LogPersonaInteraction("test-session", "greeting", "business", response)
	})
}

// Test PersonaMonitor
func TestPersonaMonitor(t *testing.T) {
	t.Run("NewPersonaMonitor", func(t *testing.T) {
		monitor := NewPersonaMonitor()
		assert.NotNil(t, monitor)
		assert.NotNil(t, monitor.metrics)
		assert.NotNil(t, monitor.logger)
		assert.Equal(t, 0, monitor.metrics.TotalInteractions)
		assert.Equal(t, 0.0, monitor.metrics.SatisfactionScore)
		assert.NotNil(t, monitor.metrics.PersonaUsage)
		assert.Empty(t, monitor.metrics.PersonaUsage)
	})

	t.Run("RecordInteraction", func(t *testing.T) {
		monitor := NewPersonaMonitor()
		
		response := &PersonalizedResponse{
			Text:      "Test response",
			Intent:    "greeting",
			Persona:   "business",
			Context:   map[string]interface{}{"test": "data"},
			Timestamp: time.Now().Unix(),
		}
		
		monitor.RecordInteraction("test-session", "greeting", "business", response, time.Second)
		
		metrics := monitor.GetMetrics()
		assert.Equal(t, 1, metrics.TotalInteractions)
		assert.Equal(t, 1, metrics.PersonaUsage["business"])
		assert.Equal(t, time.Second, metrics.AvgResponseTime)
	})

	t.Run("GetMetrics", func(t *testing.T) {
		monitor := NewPersonaMonitor()
		metrics := monitor.GetMetrics()
		
		assert.Equal(t, 0, metrics.TotalInteractions)
		assert.Equal(t, 0.0, metrics.SatisfactionScore)
		assert.NotNil(t, metrics.PersonaUsage)
		assert.Empty(t, metrics.PersonaUsage)
	})
}

// Test PersonaMetrics
func TestPersonaMetrics(t *testing.T) {
	t.Run("GetMetrics", func(t *testing.T) {
		monitor := NewPersonaMonitor()
		metrics := monitor.GetMetrics()
		
		assert.Equal(t, 0, metrics.TotalInteractions)
		assert.Equal(t, 0.0, metrics.SatisfactionScore)
		assert.NotNil(t, metrics.PersonaUsage)
		assert.Empty(t, metrics.PersonaUsage)
	})
}

// Test PersonaHealthCheck
func TestPersonaHealthCheck(t *testing.T) {
	t.Run("HealthCheck success", func(t *testing.T) {
		manager := NewPersonaManager()
		err := manager.HealthCheck()
		assert.NoError(t, err)
	})

	t.Run("HealthCheck warning", func(t *testing.T) {
		manager := NewPersonaManager()
		// Remove default persona to trigger warning
		manager.defaultPersona = nil
		err := manager.HealthCheck()
		assert.NoError(t, err) // Should not error, just log warning
	})
}

// Benchmark tests
func BenchmarkPersonaManager_GenerateSystemPrompt(b *testing.B) {
	manager := NewPersonaManager()
	cfg := Config{
		BusinessName:      "Test Business",
		BusinessHours:     "9:00-18:00",
		BusinessLanguage:  "en",
		PersonalityFriendly: true,
		PersonalityFormal: false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.GenerateSystemPrompt(cfg)
	}
}

func BenchmarkPersonalizationEngine_GeneratePersonalizedResponse(b *testing.B) {
	engine := NewPersonalizationEngine()
	cfg := Config{
		BusinessName:      "Test Business",
		BusinessHours:     "9:00-18:00",
		BusinessLanguage:  "en",
		PersonalityFriendly: true,
		PersonalityFormal: false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.GeneratePersonalizedResponse("greeting", "Hello", cfg)
	}
}

func BenchmarkTemplateManager_GetTemplate(b *testing.B) {
	manager := NewTemplateManager()
	context := map[string]interface{}{
		"time_of_day": "morning",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.GetTemplate("greeting", context)
	}
}

// Integration test for persona system
func TestPersonaSystemIntegration(t *testing.T) {
	t.Run("End-to-end persona processing", func(t *testing.T) {
		engine := NewPersonalizationEngine()
		cfg := Config{
			BusinessName:      "Acme Corp",
			BusinessHours:     "8:00-17:00",
			BusinessLanguage:  "en",
			PersonalityFriendly: true,
			PersonalityFormal: false,
		}
		
		// Test different intents
		testCases := []struct {
			intent string
			input  string
			contains string
		}{
			{"greeting", "Hello", "Welcome to Acme Corp"},
			{"goodbye", "Goodbye", "Thank you for contacting Acme Corp"},
			{"unknown", "Random text", "rephrase your question"},
			{"business_hours", "What are your hours?", "We are open from"},
			{"appointment", "Book appointment", "I can help you book"},
		}
		
		for _, tc := range testCases {
			response := engine.GeneratePersonalizedResponse(tc.intent, tc.input, cfg)
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

	t.Run("Persona system with business context", func(t *testing.T) {
		engine := NewPersonalizationEngine()
		
		// Update business context
		newContext := &BusinessContext{
			Name:        "Tech Solutions",
			Industry:    "IT Services",
			Location:    "Manila",
			Hours:       "9:00-18:00",
			Phone:       "+63 2 555 1234",
			Email:       "info@techsolutions.com",
			Website:     "https://techsolutions.com",
			Description: "Leading IT solutions provider",
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
		engine.contextManager.UpdateBusinessContext(ctx, newContext)
		
		cfg := Config{
			BusinessName:      "Tech Solutions",
			BusinessHours:     "9:00-18:00",
			BusinessLanguage:  "en",
			PersonalityFriendly: true,
			PersonalityFormal: false,
		}
		
		response := engine.GeneratePersonalizedResponse("greeting", "Hello", cfg)
		assert.NotNil(t, response)
		assert.Contains(t, response.Text, "Tech Solutions")
		
		// Check that dynamic context includes updated business info
		context := engine.contextManager.GetDynamicContext()
		assert.Equal(t, "Tech Solutions", context["business_name"])
		assert.Equal(t, "Manila", context["business_location"])
		assert.Equal(t, "Consulting", context["services"].([]string)[0])
	})

	t.Run("Persona system performance under load", func(t *testing.T) {
		engine := NewPersonalizationEngine()
		cfg := Config{
			BusinessName:      "Test Business",
			BusinessHours:     "9:00-18:00",
			BusinessLanguage:  "en",
			PersonalityFriendly: true,
			PersonalityFormal: false,
		}
		
		intents := []string{"greeting", "goodbye", "unknown", "business_hours", "appointment"}
		inputs := []string{"Hello", "Goodbye", "Random", "Hours?", "Book"}
		
		start := time.Now()
		for i := 0; i < 1000; i++ {
			intent := intents[i%len(intents)]
			input := inputs[i%len(inputs)]
			_ = engine.GeneratePersonalizedResponse(intent, input, cfg)
		}
		duration := time.Since(start)
		
		// Should process 1000 responses in under 1 second
		assert.Less(t, duration, time.Second, "Processing 1000 responses should be fast")
	})
}