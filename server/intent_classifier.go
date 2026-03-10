package main

import (
	"context"
	"log"
	"strings"
	"time"
)

// Intent represents the type of user request
type Intent string

const (
	IntentBook    Intent = "book"
	IntentInquire Intent = "inquire" 
	IntentTransfer Intent = "transfer"
	IntentUnknown Intent = "unknown"
	IntentGreeting Intent = "greeting"
	IntentGoodbye Intent = "goodbye"
)

// IntentClassifier performs intent classification using keyword matching
// In a production system, this would use a trained ML model
type IntentClassifier struct {
	keywords map[Intent][]string
}

// NewIntentClassifier creates a new intent classifier
func NewIntentClassifier() *IntentClassifier {
	return &IntentClassifier{
		keywords: map[Intent][]string{
			IntentBook: {
				"book", "reserve", "schedule", "appointment", "make reservation",
				"book appointment", "reserve slot", "schedule meeting",
			},
			IntentInquire: {
				"what", "when", "where", "how", "price", "cost", "hours",
				"information", "details", "ask", "question", "tell me",
				"business hours", "pricing", "available", "open",
			},
			IntentTransfer: {
				"transfer", "connect", "speak to", "talk to", "manager",
				"human", "representative", "agent", "live person",
				"transfer to", "connect me to",
			},
			IntentGreeting: {
				"hello", "hi", "hey", "good morning", "good afternoon", "good evening",
				"greetings", "howdy", "welcome",
			},
			IntentGoodbye: {
				"goodbye", "bye", "see you", "thank you", "thanks", "appreciate",
				"have a good day", "take care", "that's all", "nothing else",
			},
		},
	}
}

// Classify determines the intent of a given text
func (ic *IntentClassifier) Classify(text string) Intent {
	text = strings.ToLower(text)
	
	// Check for exact matches first
	for intent, keywords := range ic.keywords {
		for _, keyword := range keywords {
			if strings.Contains(text, keyword) {
				return intent
			}
		}
	}
	
	// If no match found, return unknown
	return IntentUnknown
}

// IntentHandler manages intent-specific processing
type IntentHandler struct {
	classifier *IntentClassifier
}

// NewIntentHandler creates a new intent handler
func NewIntentHandler() *IntentHandler {
	return &IntentHandler{
		classifier: NewIntentClassifier(),
	}
}

// HandleIntent processes the intent and returns appropriate response
func (ih *IntentHandler) HandleIntent(ctx context.Context, text string) (Intent, string, error) {
	intent := ih.classifier.Classify(text)
	
	switch intent {
	case IntentGreeting:
		return intent, "Hello! Welcome to " + getBusinessName() + ". How can I help you today?", nil
	case IntentGoodbye:
		return intent, "Thank you for calling " + getBusinessName() + ". Have a great day!", nil
	case IntentBook:
		return intent, "I can help you book an appointment. Let me connect you to our booking system.", nil
	case IntentInquire:
		return intent, "I'll help you find that information. Let me search our knowledge base.", nil
	case IntentTransfer:
		return intent, "I'll transfer you to a representative who can assist you further.", nil
	case IntentUnknown:
		return intent, "", nil // Let the main LLM handle unknown intents
	default:
		return intent, "", nil
	}
}

// getBusinessName retrieves the business name from environment or config
func getBusinessName() string {
	// This would typically come from the session config
	// For now, return a default
	return "Your Business Name"
}

// IntentResult contains the result of intent classification
type IntentResult struct {
	Intent    Intent  `json:"intent"`
	Confidence float64 `json:"confidence"`
	Keywords  []string `json:"keywords"`
}

// ClassifyWithDetails provides detailed classification results
func (ic *IntentClassifier) ClassifyWithDetails(text string) IntentResult {
	text = strings.ToLower(text)
	intent := IntentUnknown
	confidence := 0.0
	keywordsFound := []string{}
	
	for intentType, keywords := range ic.keywords {
		for _, keyword := range keywords {
			if strings.Contains(text, keyword) {
				if confidence < 0.8 { // Prefer more specific intents
					intent = intentType
					confidence = 0.8
				}
				keywordsFound = append(keywordsFound, keyword)
			}
		}
	}
	
	if len(keywordsFound) == 0 {
		confidence = 0.0
	} else {
		confidence = float64(len(keywordsFound)) / 5.0 // Normalize confidence
		if confidence > 1.0 {
			confidence = 1.0
		}
	}
	
	return IntentResult{
		Intent:     intent,
		Confidence: confidence,
		Keywords:   keywordsFound,
	}
}

// IntentWorkflow represents a workflow triggered by an intent
type IntentWorkflow struct {
	Intent     Intent    `json:"intent"`
	WorkflowID string    `json:"workflow_id"`
	Parameters map[string]interface{} `json:"parameters"`
	CreatedAt  time.Time `json:"created_at"`
}

// IntentWorkflowManager handles intent-based workflows
type IntentWorkflowManager struct {
	workflows map[Intent]string
}

// NewIntentWorkflowManager creates a new workflow manager
func NewIntentWorkflowManager() *IntentWorkflowManager {
	return &IntentWorkflowManager{
		workflows: map[Intent]string{
			IntentBook:    "appointment_booking",
			IntentTransfer: "transfer_to_agent",
			IntentInquire: "knowledge_search",
		},
	}
}

// GetWorkflow returns the workflow ID for a given intent
func (wm *IntentWorkflowManager) GetWorkflow(intent Intent) string {
	if workflowID, exists := wm.workflows[intent]; exists {
		return workflowID
	}
	return ""
}

// CreateWorkflow creates a new workflow instance
func (wm *IntentWorkflowManager) CreateWorkflow(intent Intent, parameters map[string]interface{}) *IntentWorkflow {
	return &IntentWorkflow{
		Intent:     intent,
		WorkflowID: wm.GetWorkflow(intent),
		Parameters: parameters,
		CreatedAt:  time.Now(),
	}
}

// IntentContext provides context for intent processing
type IntentContext struct {
	Intent       Intent                    `json:"intent"`
	Confidence   float64                   `json:"confidence"`
	Keywords     []string                  `json:"keywords"`
	Workflow     *IntentWorkflow           `json:"workflow,omitempty"`
	ShouldRoute  bool                      `json:"should_route"`
}

// ProcessIntent processes an intent and returns context
func (ih *IntentHandler) ProcessIntent(text string) *IntentContext {
	classification := ih.classifier.ClassifyWithDetails(text)
	
	workflowManager := NewIntentWorkflowManager()
	workflowID := workflowManager.GetWorkflow(classification.Intent)
	
	var workflow *IntentWorkflow
	if workflowID != "" {
		workflow = workflowManager.CreateWorkflow(classification.Intent, map[string]interface{}{
			"user_input": text,
			"timestamp":  time.Now().Unix(),
		})
	}
	
	// Determine if we should route to specialized handling
	shouldRoute := classification.Confidence > 0.5 && workflowID != ""
	
	return &IntentContext{
		Intent:     classification.Intent,
		Confidence: classification.Confidence,
		Keywords:   classification.Keywords,
		Workflow:   workflow,
		ShouldRoute: shouldRoute,
	}
}

// IntentLogger logs intent classification results
type IntentLogger struct{}

// LogIntent logs intent classification for monitoring and improvement
func (il *IntentLogger) LogIntent(sessionID string, text string, result *IntentContext) {
	log.Printf("[Intent] Session: %s, Text: %q, Intent: %s, Confidence: %.2f, Keywords: %v, ShouldRoute: %v",
		sessionID, text, result.Intent, result.Confidence, result.Keywords, result.ShouldRoute)
}

// IntentMetrics tracks intent classification metrics
type IntentMetrics struct {
	TotalClassifications int     `json:"total_classifications"`
	Accuracy             float64 `json:"accuracy"`
	TopIntents           map[string]int `json:"top_intents"`
}

// GetMetrics returns intent classification metrics
func (il *IntentLogger) GetMetrics() IntentMetrics {
	// In a real implementation, this would aggregate from a database
	return IntentMetrics{
		TotalClassifications: 0,
		Accuracy:             0.0,
		TopIntents:           make(map[string]int),
	}
}