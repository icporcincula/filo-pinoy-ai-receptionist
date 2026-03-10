package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test Intent types and constants
func TestIntentConstants(t *testing.T) {
	t.Run("Intent string values", func(t *testing.T) {
		assert.Equal(t, "book", string(IntentBook))
		assert.Equal(t, "inquire", string(IntentInquire))
		assert.Equal(t, "transfer", string(IntentTransfer))
		assert.Equal(t, "unknown", string(IntentUnknown))
		assert.Equal(t, "greeting", string(IntentGreeting))
		assert.Equal(t, "goodbye", string(IntentGoodbye))
	})
}

// Test IntentClassifier
func TestIntentClassifier(t *testing.T) {
	t.Run("NewIntentClassifier", func(t *testing.T) {
		classifier := NewIntentClassifier()
		assert.NotNil(t, classifier)
		assert.NotNil(t, classifier.keywords)
		assert.Len(t, classifier.keywords, 5) // book, inquire, transfer, greeting, goodbye
	})

	t.Run("Classify book intents", func(t *testing.T) {
		classifier := NewIntentClassifier()
		
		bookQueries := []string{
			"I want to book an appointment",
			"Can you reserve a slot for me?",
			"Schedule a meeting for next week",
			"Make a reservation for two",
			"Book appointment",
			"reserve slot",
			"schedule meeting",
		}
		
		for _, query := range bookQueries {
			intent := classifier.Classify(query)
			assert.Equal(t, IntentBook, intent, "Query: %s", query)
		}
	})

	t.Run("Classify inquire intents", func(t *testing.T) {
		classifier := NewIntentClassifier()
		
		inquireQueries := []string{
			"What are your business hours?",
			"When do you open?",
			"Where are you located?",
			"How much does it cost?",
			"What services do you offer?",
			"Tell me about your pricing",
			"Ask about your hours",
			"Question about your business",
		}
		
		for _, query := range inquireQueries {
			intent := classifier.Classify(query)
			assert.Equal(t, IntentInquire, intent, "Query: %s", query)
		}
	})

	t.Run("Classify transfer intents", func(t *testing.T) {
		classifier := NewIntentClassifier()
		
		transferQueries := []string{
			"Transfer me to a manager",
			"Connect me to a representative",
			"I want to speak to a human",
			"Talk to an agent",
			"Transfer to",
			"connect me to",
		}
		
		for _, query := range transferQueries {
			intent := classifier.Classify(query)
			assert.Equal(t, IntentTransfer, intent, "Query: %s", query)
		}
	})

	t.Run("Classify greeting intents", func(t *testing.T) {
		classifier := NewIntentClassifier()
		
		greetingQueries := []string{
			"Hello",
			"Hi there",
			"Good morning",
			"Good afternoon",
			"Good evening",
			"Welcome",
			"Howdy",
		}
		
		for _, query := range greetingQueries {
			intent := classifier.Classify(query)
			assert.Equal(t, IntentGreeting, intent, "Query: %s", query)
		}
	})

	t.Run("Classify goodbye intents", func(t *testing.T) {
		classifier := NewIntentClassifier()
		
		goodbyeQueries := []string{
			"Goodbye",
			"Bye",
			"See you later",
			"Thank you",
			"Thanks",
			"Appreciate your help",
			"Have a good day",
			"Take care",
			"That's all",
			"Nothing else",
		}
		
		for _, query := range goodbyeQueries {
			intent := classifier.Classify(query)
			assert.Equal(t, IntentGoodbye, intent, "Query: %s", query)
		}
	})

	t.Run("Classify unknown intents", func(t *testing.T) {
		classifier := NewIntentClassifier()
		
		unknownQueries := []string{
			"Random text with no keywords",
			"This is just some random sentence",
			"Completely unrelated content",
			"Testing unknown classification",
		}
		
		for _, query := range unknownQueries {
			intent := classifier.Classify(query)
			assert.Equal(t, IntentUnknown, intent, "Query: %s", query)
		}
	})

	t.Run("Case insensitive classification", func(t *testing.T) {
		classifier := NewIntentClassifier()
		
		testCases := []struct {
			input    string
			expected Intent
		}{
			{"BOOK APPOINTMENT", IntentBook},
			{"RESERVE SLOT", IntentBook},
			{"WHAT ARE YOUR HOURS?", IntentInquire},
			{"WHERE ARE YOU LOCATED?", IntentInquire},
			{"TRANSFER TO MANAGER", IntentTransfer},
			{"CONNECT TO REPRESENTATIVE", IntentTransfer},
			{"HELLO", IntentGreeting},
			{"GOODBYE", IntentGoodbye},
			{"THANK YOU", IntentGoodbye},
		}
		
		for _, tc := range testCases {
			intent := classifier.Classify(tc.input)
			assert.Equal(t, tc.expected, intent, "Input: %s", tc.input)
		}
	})

	t.Run("Partial keyword matching", func(t *testing.T) {
		classifier := NewIntentClassifier()
		
		// Test that partial matches work
		intent := classifier.Classify("I want to book")
		assert.Equal(t, IntentBook, intent)
		
		intent = classifier.Classify("Can you tell me")
		assert.Equal(t, IntentInquire, intent)
		
		intent = classifier.Classify("Transfer please")
		assert.Equal(t, IntentTransfer, intent)
	})
}

// Test IntentHandler
func TestIntentHandler(t *testing.T) {
	t.Run("NewIntentHandler", func(t *testing.T) {
		handler := NewIntentHandler()
		assert.NotNil(t, handler)
		assert.NotNil(t, handler.classifier)
	})

	t.Run("HandleIntent", func(t *testing.T) {
		handler := NewIntentHandler()
		ctx := context.Background()
		
		testCases := []struct {
			input    string
			expected Intent
			contains string
		}{
			{"Hello", IntentGreeting, "Welcome to"},
			{"What are your hours?", IntentInquire, "search our knowledge base"},
			{"Book an appointment", IntentBook, "booking system"},
			{"Transfer to manager", IntentTransfer, "representative"},
			{"Thank you", IntentGoodbye, "Have a great day"},
			{"Random text", IntentUnknown, ""},
		}
		
		for _, tc := range testCases {
			intent, response, err := handler.HandleIntent(ctx, tc.input)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, intent)
			
			if tc.contains != "" {
				assert.Contains(t, response, tc.contains)
			}
		}
	})
}

// Test IntentResult and detailed classification
func TestIntentClassificationDetails(t *testing.T) {
	t.Run("ClassifyWithDetails", func(t *testing.T) {
		classifier := NewIntentClassifier()
		
		// Test book intent with details
		result := classifier.ClassifyWithDetails("I want to book an appointment")
		assert.Equal(t, IntentBook, result.Intent)
		assert.Greater(t, result.Confidence, 0.0)
		assert.Contains(t, result.Keywords, "book")
		
		// Test inquire intent with details
		result = classifier.ClassifyWithDetails("What are your business hours?")
		assert.Equal(t, IntentInquire, result.Intent)
		assert.Greater(t, result.Confidence, 0.0)
		assert.Contains(t, result.Keywords, "what")
		
		// Test unknown intent
		result = classifier.ClassifyWithDetails("Random text")
		assert.Equal(t, IntentUnknown, result.Intent)
		assert.Equal(t, 0.0, result.Confidence)
		assert.Empty(t, result.Keywords)
	})

	t.Run("Multiple keywords detection", func(t *testing.T) {
		classifier := NewIntentClassifier()
		
		// Text with multiple keywords
		result := classifier.ClassifyWithDetails("I want to book and schedule an appointment")
		assert.Equal(t, IntentBook, result.Intent)
		assert.Greater(t, result.Confidence, 0.0)
		assert.Contains(t, result.Keywords, "book")
		assert.Contains(t, result.Keywords, "schedule")
	})
}

// Test IntentWorkflow
func TestIntentWorkflow(t *testing.T) {
	t.Run("NewIntentWorkflowManager", func(t *testing.T) {
		manager := NewIntentWorkflowManager()
		assert.NotNil(t, manager)
		assert.NotNil(t, manager.workflows)
		assert.Len(t, manager.workflows, 3) // book, transfer, inquire
	})

	t.Run("GetWorkflow", func(t *testing.T) {
		manager := NewIntentWorkflowManager()
		
		assert.Equal(t, "appointment_booking", manager.GetWorkflow(IntentBook))
		assert.Equal(t, "transfer_to_agent", manager.GetWorkflow(IntentTransfer))
		assert.Equal(t, "knowledge_search", manager.GetWorkflow(IntentInquire))
		assert.Equal(t, "", manager.GetWorkflow(IntentUnknown))
	})

	t.Run("CreateWorkflow", func(t *testing.T) {
		manager := NewIntentWorkflowManager()
		
		parameters := map[string]interface{}{
			"user_input": "Test input",
			"timestamp":  1234567890,
		}
		
		workflow := manager.CreateWorkflow(IntentBook, parameters)
		assert.NotNil(t, workflow)
		assert.Equal(t, IntentBook, workflow.Intent)
		assert.Equal(t, "appointment_booking", workflow.WorkflowID)
		assert.Equal(t, parameters, workflow.Parameters)
		assert.NotZero(t, workflow.CreatedAt)
	})
}

// Test IntentContext
func TestIntentContext(t *testing.T) {
	t.Run("ProcessIntent", func(t *testing.T) {
		handler := NewIntentHandler()
		
		// Test book intent
		context := handler.ProcessIntent("I want to book an appointment")
		assert.NotNil(t, context)
		assert.Equal(t, IntentBook, context.Intent)
		assert.Greater(t, context.Confidence, 0.0)
		assert.Contains(t, context.Keywords, "book")
		assert.True(t, context.ShouldRoute)
		
		// Test unknown intent
		context = handler.ProcessIntent("Random text")
		assert.NotNil(t, context)
		assert.Equal(t, IntentUnknown, context.Intent)
		assert.Equal(t, 0.0, context.Confidence)
		assert.Empty(t, context.Keywords)
		assert.False(t, context.ShouldRoute)
	})
}

// Test IntentLogger
func TestIntentLogger(t *testing.T) {
	t.Run("LogIntent", func(t *testing.T) {
		logger := &IntentLogger{}
		
		context := &IntentContext{
			Intent:     IntentBook,
			Confidence: 0.8,
			Keywords:   []string{"book"},
			ShouldRoute: true,
		}
		
		// This should not panic and should log the intent
		logger.LogIntent("test-session", "I want to book", context)
	})
}

// Test IntentMetrics
func TestIntentMetrics(t *testing.T) {
	t.Run("GetMetrics", func(t *testing.T) {
		logger := &IntentLogger{}
		metrics := logger.GetMetrics()
		
		assert.Equal(t, 0, metrics.TotalClassifications)
		assert.Equal(t, 0.0, metrics.Accuracy)
		assert.NotNil(t, metrics.TopIntents)
		assert.Empty(t, metrics.TopIntents)
	})
}

// Benchmark tests
func BenchmarkIntentClassifier_Classify(b *testing.B) {
	classifier := NewIntentClassifier()
	queries := []string{
		"I want to book an appointment",
		"What are your business hours?",
		"Transfer me to a manager",
		"Hello",
		"Thank you",
		"Random text with no keywords",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		_ = classifier.Classify(query)
	}
}

func BenchmarkIntentHandler_HandleIntent(b *testing.B) {
	handler := NewIntentHandler()
	ctx := context.Background()
	queries := []string{
		"I want to book an appointment",
		"What are your business hours?",
		"Transfer me to a manager",
		"Hello",
		"Thank you",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		_, _, _ = handler.HandleIntent(ctx, query)
	}
}

// Integration test for intent classification pipeline
func TestIntentClassificationPipeline(t *testing.T) {
	t.Run("End-to-end intent processing", func(t *testing.T) {
		handler := NewIntentHandler()
		ctx := context.Background()
		
		// Test a complete conversation flow
		testQueries := []struct {
			query    string
			expected Intent
		}{
			{"Hello, how are you?", IntentGreeting},
			{"What time do you open?", IntentInquire},
			{"I'd like to book an appointment", IntentBook},
			{"Can you transfer me to someone?", IntentTransfer},
			{"Thank you for your help", IntentGoodbye},
		}
		
		for _, tc := range testQueries {
			context := handler.ProcessIntent(tc.query)
			assert.Equal(t, tc.expected, context.Intent, "Query: %s", tc.query)
			
			// Test that the handler can process the intent
			intent, response, err := handler.HandleIntent(ctx, tc.query)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, intent)
			
			// For known intents, we should get a response
			if tc.expected != IntentUnknown {
				assert.NotEmpty(t, response)
			}
		}
	})

	t.Run("Performance under load", func(t *testing.T) {
		handler := NewIntentHandler()
		ctx := context.Background()
		
		// Simulate high volume of requests
		queries := []string{
			"I want to book an appointment",
			"What are your business hours?",
			"Transfer me to a manager",
			"Hello",
			"Thank you",
			"Random text with no keywords",
		}
		
		start := time.Now()
		for i := 0; i < 1000; i++ {
			query := queries[i%len(queries)]
			_, _, _ = handler.HandleIntent(ctx, query)
		}
		duration := time.Since(start)
		
		// Should process 1000 intents in under 1 second
		assert.Less(t, duration, time.Second, "Processing 1000 intents should be fast")
	})
}