package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// Persona represents a business persona configuration
type Persona struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Tone        string            `json:"tone"`
	Language    string            `json:"language"`
	Formality   string            `json:"formality"`
	Keywords    []string          `json:"keywords"`
	Responses   map[string]string `json:"responses"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// PersonaManager manages business personas and dynamic system prompts
type PersonaManager struct {
	personas map[string]*Persona
	defaultPersona *Persona
}

// NewPersonaManager creates a new persona manager
func NewPersonaManager() *PersonaManager {
	return &PersonaManager{
		personas: make(map[string]*Persona),
		defaultPersona: &Persona{
			Name:        "Default",
			Description: "Default business assistant persona",
			Tone:        "professional",
			Language:    "en",
			Formality:   "neutral",
			Keywords:    []string{"help", "assistance", "support"},
			Responses: map[string]string{
				"greeting": "Hello! How can I help you today?",
				"goodbye":  "Thank you for contacting us. Have a great day!",
				"unknown":  "I'm not sure I understand. Could you please rephrase your question?",
			},
			Metadata: map[string]interface{}{
				"created_at": time.Now().Unix(),
				"version":    "1.0",
			},
		},
	}
}

// AddPersona adds a new persona to the manager
func (pm *PersonaManager) AddPersona(persona *Persona) {
	pm.personas[persona.Name] = persona
}

// GetPersona retrieves a persona by name
func (pm *PersonaManager) GetPersona(name string) *Persona {
	if persona, exists := pm.personas[name]; exists {
		return persona
	}
	return pm.defaultPersona
}

// GetDefaultPersona returns the default persona
func (pm *PersonaManager) GetDefaultPersona() *Persona {
	return pm.defaultPersona
}

// GenerateSystemPrompt generates a dynamic system prompt based on persona and business context
func (pm *PersonaManager) GenerateSystemPrompt(cfg Config) string {
	persona := pm.GetPersona("business") // Use business persona if available
	var systemPrompt string
	
	// Build the system prompt
	var promptBuilder strings.Builder
	
	// Start with base instruction
	promptBuilder.WriteString("You are a ")
	
	// Add formality level
	switch cfg.PersonalityFormal {
	case true:
		promptBuilder.WriteString("formal and professional ")
	case false:
		promptBuilder.WriteString("friendly and conversational ")
	}
	
	// Add business context
	promptBuilder.WriteString("voice assistant for ")
	promptBuilder.WriteString(cfg.BusinessName)
	promptBuilder.WriteString(". ")
	
	// Add business hours context
	if cfg.BusinessHours != "" {
		promptBuilder.WriteString("Business hours are ")
		promptBuilder.WriteString(cfg.BusinessHours)
		promptBuilder.WriteString(". ")
	}
	
	// Add language preference
	if cfg.BusinessLanguage != "" && cfg.BusinessLanguage != "en" {
		promptBuilder.WriteString("Respond in ")
		promptBuilder.WriteString(cfg.BusinessLanguage)
		promptBuilder.WriteString(". ")
	}
	
	// Add tone and style
	if cfg.PersonalityFriendly {
		promptBuilder.WriteString("Be helpful, friendly, and personable. ")
	} else {
		promptBuilder.WriteString("Be professional and to the point. ")
	}
	
	// Add response style
	promptBuilder.WriteString("Keep responses short and conversational. ")
	promptBuilder.WriteString("Avoid markdown formatting or bullet points. ")
	
	// Add business-specific instructions
	promptBuilder.WriteString("Handle common business requests like booking appointments, ")
	promptBuilder.WriteString("answering questions about services, and transferring to representatives. ")
	
	// Add persona-specific instructions
	if len(persona.Keywords) > 0 {
		promptBuilder.WriteString("Use keywords: ")
		promptBuilder.WriteString(strings.Join(persona.Keywords, ", "))
		promptBuilder.WriteString(". ")
	}
	
	// Add closing instruction
	promptBuilder.WriteString("Always maintain a helpful and professional demeanor.")
	
	return promptBuilder.String()
}

// BusinessContext contains business-specific information
type BusinessContext struct {
	Name        string    `json:"name"`
	Industry    string    `json:"industry"`
	Location    string    `json:"location"`
	Hours       string    `json:"hours"`
	Phone       string    `json:"phone"`
	Email       string    `json:"email"`
	Website     string    `json:"website"`
	Description string    `json:"description"`
	Services    []string  `json:"services"`
	FAQ         []FAQItem `json:"faq"`
}

// FAQItem represents a frequently asked question
type FAQItem struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Category string `json:"category"`
}

// ContextManager manages business context and dynamic information
type ContextManager struct {
	businessContext *BusinessContext
	personaManager  *PersonaManager
}

// NewContextManager creates a new context manager
func NewContextManager() *ContextManager {
	return &ContextManager{
		businessContext: &BusinessContext{
			Name:        "Your Business Name",
			Industry:    "Service",
			Location:    "Philippines",
			Hours:       "9:00 AM - 6:00 PM",
			Phone:       "+63 2 123 4567",
			Email:       "info@yourbusiness.com",
			Website:     "https://yourbusiness.com",
			Description: "We provide excellent service to our customers.",
			Services:    []string{"Consultation", "Support", "Training"},
			FAQ: []FAQItem{
				{
					Question: "What are your business hours?",
					Answer:   "We are open from 9:00 AM to 6:00 PM, Monday through Friday.",
					Category: "General",
				},
				{
					Question: "How can I book an appointment?",
					Answer:   "You can book an appointment by speaking to me, and I'll help you find available time slots.",
					Category: "Services",
				},
			},
		},
		personaManager: NewPersonaManager(),
	}
}

// UpdateBusinessContext updates the business context
func (cm *ContextManager) UpdateBusinessContext(ctx context.Context, businessContext *BusinessContext) {
	cm.businessContext = businessContext
}

// GetBusinessContext returns the current business context
func (cm *ContextManager) GetBusinessContext() *BusinessContext {
	return cm.businessContext
}

// GetDynamicContext returns dynamic context for LLM processing
func (cm *ContextManager) GetDynamicContext() map[string]interface{} {
	return map[string]interface{}{
		"business_name":     cm.businessContext.Name,
		"business_hours":    cm.businessContext.Hours,
		"business_location": cm.businessContext.Location,
		"current_time":      time.Now().Format("2006-01-02 15:04:05"),
		"day_of_week":       time.Now().Weekday().String(),
		"services":          cm.businessContext.Services,
		"faq_count":         len(cm.businessContext.FAQ),
	}
}

// ResponseTemplate represents a templated response
type ResponseTemplate struct {
	Intent      string `json:"intent"`
	Template    string `json:"template"`
	Priority    int    `json:"priority"`
	Conditions  map[string]interface{} `json:"conditions"`
}

// TemplateManager manages response templates
type TemplateManager struct {
	templates map[string][]*ResponseTemplate
}

// NewTemplateManager creates a new template manager
func NewTemplateManager() *TemplateManager {
	return &TemplateManager{
		templates: map[string][]*ResponseTemplate{
			"greeting": {
				{
					Intent:   "greeting",
					Template: "Hello! Welcome to {{business_name}}. How can I help you today?",
					Priority: 1,
					Conditions: map[string]interface{}{
						"time_of_day": "morning",
					},
				},
				{
					Intent:   "greeting",
					Template: "Hi there! Welcome to {{business_name}}. How can I assist you?",
					Priority: 2,
					Conditions: map[string]interface{}{
						"time_of_day": "afternoon",
					},
				},
			},
			"goodbye": {
				{
					Intent:   "goodbye",
					Template: "Thank you for contacting {{business_name}}. Have a wonderful day!",
					Priority: 1,
					Conditions: map[string]interface{}{},
				},
			},
			"unknown": {
				{
					Intent:   "unknown",
					Template: "I'm not sure I understand. Could you please rephrase your question? Or would you like me to transfer you to a representative?",
					Priority: 1,
					Conditions: map[string]interface{}{},
				},
			},
			"business_hours": {
				{
					Intent:   "business_hours",
					Template: "We are open from {{business_hours}}. Is there anything I can help you with during our business hours?",
					Priority: 1,
					Conditions: map[string]interface{}{},
				},
			},
			"appointment": {
				{
					Intent:   "appointment",
					Template: "I can help you book an appointment. Let me check our availability for you.",
					Priority: 1,
					Conditions: map[string]interface{}{},
				},
			},
		},
	}
}

// GetTemplate retrieves the best matching template for an intent
func (tm *TemplateManager) GetTemplate(intent string, context map[string]interface{}) string {
	templates, exists := tm.templates[intent]
	if !exists || len(templates) == 0 {
		return ""
	}
	
	// Find the highest priority template that matches conditions
	var bestTemplate *ResponseTemplate
	bestPriority := -1
	
	for _, template := range templates {
		if template.Priority > bestPriority {
			// Check if template conditions are met
			ifmatchConditions(template.Conditions, context) {
				bestTemplate = template
				bestPriority = template.Priority
			}
		}
	}
	
	if bestTemplate != nil {
		return bestTemplate.Template
	}
	
	// If no specific template found, try generic templates
	genericTemplates, exists := tm.templates["generic"]
	if exists && len(genericTemplates) > 0 {
		return genericTemplates[0].Template
	}
	
	return ""
}

//matchConditions checks if template conditions are met
funcmatchConditions(conditions map[string]interface{}, context map[string]interface{}) bool {
	if len(conditions) == 0 {
		return true
	}
	
	for key, value := range conditions {
		if contextValue, exists := context[key]; exists {
			if contextValue != value {
				return false
			}
		} else {
			return false
		}
	}
	
	return true
}

// PersonalizedResponse generates a personalized response
type PersonalizedResponse struct {
	Text      string                 `json:"text"`
	Intent    string                 `json:"intent"`
	Persona   string                 `json:"persona"`
	Context   map[string]interface{} `json:"context"`
	Timestamp int64                  `json:"timestamp"`
}

// PersonalizationEngine handles response personalization
type PersonalizationEngine struct {
	contextManager *ContextManager
	templateManager *TemplateManager
	personaManager *PersonaManager
}

// NewPersonalizationEngine creates a new personalization engine
func NewPersonalizationEngine() *PersonalizationEngine {
	return &PersonalizationEngine{
		contextManager:  NewContextManager(),
		templateManager: NewTemplateManager(),
		personaManager:  NewPersonaManager(),
	}
}

// GeneratePersonalizedResponse generates a personalized response
func (pe *PersonalizationEngine) GeneratePersonalizedResponse(intent string, userInput string, cfg Config) *PersonalizedResponse {
	// Get dynamic context
	context := pe.contextManager.GetDynamicContext()
	
	// Generate system prompt
	pe.personaManager.GenerateSystemPrompt(cfg)
	
	// Get template
	template := pe.templateManager.GetTemplate(intent, context)
	
	// Build response
	var responseText string
	if template != "" {
		responseText = pe.fillTemplate(template, context)
	} else {
		// Use default responses
		persona := pe.personaManager.GetPersona("business")
		if response, exists := persona.Responses[intent]; exists {
			responseText = pe.fillTemplate(response, context)
		} else {
			responseText = "I'm here to help you. How can I assist you today?"
		}
	}
	
	return &PersonalizedResponse{
		Text:      responseText,
		Intent:    intent,
		Persona:   "business",
		Context:   context,
		Timestamp: time.Now().Unix(),
	}
}

// fillTemplate fills template variables with context values
func (pe *PersonalizationEngine) fillTemplate(template string, context map[string]interface{}) string {
	result := template
	
	// Replace template variables
	for key, value := range context {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	
	return result
}

// PersonaLogger logs persona-related operations
type PersonaLogger struct{}

// LogPersonaInteraction logs persona interactions
func (pl *PersonaLogger) LogPersonaInteraction(sessionID, intent, persona string, response *PersonalizedResponse) {
	log.Printf("[Persona] Session: %s, Intent: %s, Persona: %s, Response: %q",
		sessionID, intent, persona, response.Text)
}

// PersonaMetrics tracks persona usage metrics
type PersonaMetrics struct {
	TotalInteractions int                    `json:"total_interactions"`
	PersonaUsage      map[string]int         `json:"persona_usage"`
	AvgResponseTime   time.Duration          `json:"avg_response_time"`
	SatisfactionScore float64               `json:"satisfaction_score"`
}

// PersonaMonitor monitors persona usage and performance
type PersonaMonitor struct {
	metrics *PersonaMetrics
	logger  *PersonaLogger
}

// NewPersonaMonitor creates a new persona monitor
func NewPersonaMonitor() *PersonaMonitor {
	return &PersonaMonitor{
		metrics: &PersonaMetrics{
			TotalInteractions: 0,
			PersonaUsage:      make(map[string]int),
			AvgResponseTime:   0,
			SatisfactionScore: 0.0,
		},
		logger: &PersonaLogger{},
	}
}

// RecordInteraction records a persona interaction
func (pm *PersonaMonitor) RecordInteraction(sessionID, intent, persona string, response *PersonalizedResponse, responseTime time.Duration) {
	pm.metrics.TotalInteractions++
	pm.metrics.PersonaUsage[persona]++
	pm.logger.LogPersonaInteraction(sessionID, intent, persona, response)
}

// GetMetrics returns persona metrics
func (pm *PersonaMonitor) GetMetrics() *PersonaMetrics {
	return pm.metrics
}

// HealthCheck checks if the persona system is healthy
func (pm *PersonaManager) HealthCheck() error {
	if pm.defaultPersona == nil {
		return fmt.Errorf("default persona is not configured")
	}
	
	if len(pm.personas) == 0 {
		log.Printf("WARNING: No custom personas configured, using default only")
	}
	
	return nil
}