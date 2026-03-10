package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock HTTP client for testing
type MockWorkflowHTTPClient struct {
	mock.Mock
}

func (m *MockWorkflowHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

// Test WorkflowManager
func TestWorkflowManager(t *testing.T) {
	t.Run("NewWorkflowManager", func(t *testing.T) {
		manager := NewWorkflowManager("http://localhost:5678", "testuser", "testpass")
		assert.NotNil(t, manager)
		assert.Equal(t, "http://localhost:5678", manager.baseURL)
		assert.Equal(t, "testuser", manager.credentials.Username)
		assert.Equal(t, "testpass", manager.credentials.Password)
		assert.NotNil(t, manager.httpClient)
	})

	t.Run("TriggerWorkflow", func(t *testing.T) {
		// Create mock server
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/webhook/appointment-booking-workflow", r.URL.Path)
			
			// Verify authentication
			username, password, ok := r.BasicAuth()
			assert.True(t, ok)
			assert.Equal(t, "testuser", username)
			assert.Equal(t, "testpass", password)
			
			// Verify request body
			body, _ := io.ReadAll(r.Body)
			var trigger WorkflowTrigger
			err := json.Unmarshal(body, &trigger)
			assert.NoError(t, err)
			assert.Equal(t, "appointment-booking-workflow", trigger.WorkflowID)
			assert.NotNil(t, trigger.Input)
			
			// Return success response
			response := WorkflowResponse{
				Success: true,
				Data: map[string]interface{}{
					"appointment_id": "12345",
					"status": "confirmed",
				},
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer mockServer.Close()

		manager := NewWorkflowManager(mockServer.URL, "testuser", "testpass")
		ctx := context.Background()

		input := map[string]interface{}{
			"session_id": "test-session",
			"user_input": "Book an appointment",
			"intent":     "book",
		}

		resp, err := manager.TriggerWorkflow(ctx, "appointment-booking-workflow", input)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.NotNil(t, resp.Data)
		assert.Equal(t, "12345", resp.Data["appointment_id"])
		assert.Equal(t, "confirmed", resp.Data["status"])
	})

	t.Run("TriggerWorkflow with error", func(t *testing.T) {
		// Create mock server that returns error
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"success": false, "error": "Invalid workflow ID"}`))
		}))
		defer mockServer.Close()

		manager := NewWorkflowManager(mockServer.URL, "testuser", "testpass")
		ctx := context.Background()

		input := map[string]interface{}{}
		resp, err := manager.TriggerWorkflow(ctx, "invalid-workflow", input)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.False(t, resp.Success)
		assert.Equal(t, "Invalid workflow ID", resp.Error)
	})

	t.Run("ListWorkflows", func(t *testing.T) {
		// Create mock server
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/rest/workflows", r.URL.Path)
			
			// Verify authentication
			username, password, ok := r.BasicAuth()
			assert.True(t, ok)
			assert.Equal(t, "testuser", username)
			assert.Equal(t, "testpass", password)
			
			// Return workflows
			workflows := []map[string]interface{}{
				{
					"id":   "1",
					"name": "appointment-booking-workflow",
					"active": true,
				},
				{
					"id":   "2",
					"name": "transfer-to-agent-workflow",
					"active": true,
				},
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(workflows)
		}))
		defer mockServer.Close()

		manager := NewWorkflowManager(mockServer.URL, "testuser", "testpass")
		ctx := context.Background()

		workflows, err := manager.ListWorkflows(ctx)
		assert.NoError(t, err)
		assert.Len(t, workflows, 2)
		assert.Equal(t, "appointment-booking-workflow", workflows[0]["name"])
		assert.Equal(t, "transfer-to-agent-workflow", workflows[1]["name"])
	})

	t.Run("GetWorkflow", func(t *testing.T) {
		// Create mock server
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/rest/workflows/appointment-booking-workflow", r.URL.Path)
			
			// Return workflow details
			workflow := map[string]interface{}{
				"id":   "1",
				"name": "appointment-booking-workflow",
				"active": true,
				"nodes": []map[string]interface{}{
					{
						"id": "1",
						"type": "n8n-nodes-base.start",
					},
				},
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(workflow)
		}))
		defer mockServer.Close()

		manager := NewWorkflowManager(mockServer.URL, "testuser", "testpass")
		ctx := context.Background()

		workflow, err := manager.GetWorkflow(ctx, "appointment-booking-workflow")
		assert.NoError(t, err)
		assert.NotNil(t, workflow)
		assert.Equal(t, "appointment-booking-workflow", workflow["name"])
		assert.True(t, workflow["active"].(bool))
	})
}

// Test WorkflowRegistry
func TestWorkflowRegistry(t *testing.T) {
	t.Run("NewWorkflowRegistry", func(t *testing.T) {
		manager := NewWorkflowManager("http://localhost:5678", "testuser", "testpass")
		registry := NewWorkflowRegistry(manager)
		assert.NotNil(t, registry)
		assert.NotNil(t, registry.workflows)
		assert.NotNil(t, registry.manager)
		assert.Len(t, registry.workflows, 3) // appointment_booking, transfer_to_agent, knowledge_search
	})

	t.Run("RegisterWorkflow", func(t *testing.T) {
		manager := NewWorkflowManager("http://localhost:5678", "testuser", "testpass")
		registry := NewWorkflowRegistry(manager)
		
		registry.RegisterWorkflow("custom_intent", "custom-workflow-id")
		assert.Equal(t, "custom-workflow-id", registry.GetWorkflowID("custom_intent"))
	})

	t.Run("GetWorkflowID", func(t *testing.T) {
		manager := NewWorkflowManager("http://localhost:5678", "testuser", "testpass")
		registry := NewWorkflowRegistry(manager)
		
		assert.Equal(t, "appointment-booking-workflow", registry.GetWorkflowID("appointment_booking"))
		assert.Equal(t, "transfer-to-agent-workflow", registry.GetWorkflowID("transfer_to_agent"))
		assert.Equal(t, "knowledge-search-workflow", registry.GetWorkflowID("knowledge_search"))
		assert.Equal(t, "", registry.GetWorkflowID("unknown_intent"))
	})

	t.Run("ExecuteWorkflow", func(t *testing.T) {
		// Create mock server
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := WorkflowResponse{
				Success: true,
				Data: map[string]interface{}{
					"result": "success",
				},
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer mockServer.Close()

		manager := NewWorkflowManager(mockServer.URL, "testuser", "testpass")
		registry := NewWorkflowRegistry(manager)
		ctx := context.Background()

		input := map[string]interface{}{
			"test": "data",
		}

		resp, err := registry.ExecuteWorkflow(ctx, "appointment_booking", input)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.NotNil(t, resp.Data)
	})

	t.Run("ExecuteWorkflow with unknown intent", func(t *testing.T) {
		manager := NewWorkflowManager("http://localhost:5678", "testuser", "testpass")
		registry := NewWorkflowRegistry(manager)
		ctx := context.Background()

		input := map[string]interface{}{}
		resp, err := registry.ExecuteWorkflow(ctx, "unknown_intent", input)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "no workflow registered for intent")
	})
}

// Test WorkflowExecutor
func TestWorkflowExecutor(t *testing.T) {
	t.Run("NewWorkflowExecutor", func(t *testing.T) {
		manager := NewWorkflowManager("http://localhost:5678", "testuser", "testpass")
		executor := NewWorkflowExecutor(manager)
		assert.NotNil(t, executor)
		assert.NotNil(t, executor.registry)
		assert.NotNil(t, executor.monitor)
		assert.NotNil(t, executor.logger)
	})

	t.Run("ExecuteIntentWorkflow", func(t *testing.T) {
		// Create mock server
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := WorkflowResponse{
				Success: true,
				Data: map[string]interface{}{
					"appointment_id": "12345",
					"status": "confirmed",
				},
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer mockServer.Close()

		manager := NewWorkflowManager(mockServer.URL, "testuser", "testpass")
		executor := NewWorkflowExecutor(manager)
		ctx := context.Background()

		resp, err := executor.ExecuteIntentWorkflow(ctx, "test-session", IntentBook, "Book an appointment")
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.NotNil(t, resp.Data)
		assert.Equal(t, "12345", resp.Data["appointment_id"])
	})

	t.Run("ExecuteIntentWorkflow with error", func(t *testing.T) {
		// Create mock server that returns error
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"success": false, "error": "Workflow execution failed"}`))
		}))
		defer mockServer.Close()

		manager := NewWorkflowManager(mockServer.URL, "testuser", "testpass")
		executor := NewWorkflowExecutor(manager)
		ctx := context.Background()

		resp, err := executor.ExecuteIntentWorkflow(ctx, "test-session", IntentBook, "Book an appointment")
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "workflow execution failed")
	})
}

// Test WorkflowLogger
func TestWorkflowLogger(t *testing.T) {
	t.Run("LogWorkflowExecution", func(t *testing.T) {
		logger := &WorkflowLogger{}
		
		input := map[string]interface{}{
			"session_id": "test-session",
			"user_input": "Book an appointment",
		}
		output := map[string]interface{}{
			"appointment_id": "12345",
		}
		
		// This should not panic and should log the execution
		logger.LogWorkflowExecution("test-session", "book", "appointment-booking-workflow", input, output, true)
	})
}

// Test WorkflowMonitor
func TestWorkflowMonitor(t *testing.T) {
	t.Run("NewWorkflowMonitor", func(t *testing.T) {
		monitor := NewWorkflowMonitor()
		assert.NotNil(t, monitor)
		assert.NotNil(t, monitor.errors)
		assert.NotNil(t, monitor.logger)
		assert.Empty(t, monitor.errors)
	})

	t.Run("RecordError", func(t *testing.T) {
		monitor := NewWorkflowMonitor()
		
		monitor.RecordError("test-workflow", "Test error message")
		errors := monitor.GetErrors()
		assert.Len(t, errors, 1)
		assert.Equal(t, "test-workflow", errors[0].WorkflowID)
		assert.Equal(t, "Test error message", errors[0].Error)
		assert.NotZero(t, errors[0].Timestamp)
	})

	t.Run("ClearErrors", func(t *testing.T) {
		monitor := NewWorkflowMonitor()
		
		monitor.RecordError("test-workflow", "Test error message")
		assert.Len(t, monitor.GetErrors(), 1)
		
		monitor.ClearErrors()
		assert.Empty(t, monitor.GetErrors())
	})

	t.Run("GetMetrics", func(t *testing.T) {
		monitor := NewWorkflowMonitor()
		
		metrics := monitor.GetMetrics()
		assert.Equal(t, 0, metrics.TotalExecutions)
		assert.Equal(t, 0.0, metrics.SuccessRate)
		assert.Equal(t, time.Duration(0), metrics.AverageDuration)
		assert.NotNil(t, metrics.TopWorkflows)
		assert.Empty(t, metrics.TopWorkflows)
		assert.Equal(t, 0, metrics.ErrorCount)
	})
}

// Test WorkflowMetrics
func TestWorkflowMetrics(t *testing.T) {
	t.Run("GetMetrics", func(t *testing.T) {
		monitor := NewWorkflowMonitor()
		metrics := monitor.GetMetrics()
		
		assert.Equal(t, 0, metrics.TotalExecutions)
		assert.Equal(t, 0.0, metrics.SuccessRate)
		assert.Equal(t, time.Duration(0), metrics.AverageDuration)
		assert.NotNil(t, metrics.TopWorkflows)
		assert.Empty(t, metrics.TopWorkflows)
		assert.Equal(t, 0, metrics.ErrorCount)
	})
}

// Test WorkflowError
func TestWorkflowError(t *testing.T) {
	t.Run("WorkflowError creation", func(t *testing.T) {
		timestamp := time.Now().Unix()
		error := WorkflowError{
			WorkflowID: "test-workflow",
			Error:      "Test error",
			Timestamp:  timestamp,
		}
		
		assert.Equal(t, "test-workflow", error.WorkflowID)
		assert.Equal(t, "Test error", error.Error)
		assert.Equal(t, timestamp, error.Timestamp)
	})
}

// Benchmark tests
func BenchmarkWorkflowManager_TriggerWorkflow(b *testing.B) {
	// Create mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := WorkflowResponse{
			Success: true,
			Data: map[string]interface{}{
				"result": "success",
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	manager := NewWorkflowManager(mockServer.URL, "testuser", "testpass")
	ctx := context.Background()
	input := map[string]interface{}{
		"test": "data",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.TriggerWorkflow(ctx, "appointment-booking-workflow", input)
	}
}

func BenchmarkWorkflowRegistry_ExecuteWorkflow(b *testing.B) {
	// Create mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := WorkflowResponse{
			Success: true,
			Data: map[string]interface{}{
				"result": "success",
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	manager := NewWorkflowManager(mockServer.URL, "testuser", "testpass")
	registry := NewWorkflowRegistry(manager)
	ctx := context.Background()
	input := map[string]interface{}{
		"test": "data",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = registry.ExecuteWorkflow(ctx, "appointment_booking", input)
	}
}

// Integration test for workflow system
func TestWorkflowSystemIntegration(t *testing.T) {
	t.Run("End-to-end workflow execution", func(t *testing.T) {
		// Create mock server that simulates a complete workflow execution
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate different responses based on the workflow ID
			if r.URL.Path == "/webhook/appointment-booking-workflow" {
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
			} else if r.URL.Path == "/webhook/transfer-to-agent-workflow" {
				response := WorkflowResponse{
					Success: true,
					Data: map[string]interface{}{
						"agent_id": "agent-123",
						"queue_position": 1,
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer mockServer.Close()

		manager := NewWorkflowManager(mockServer.URL, "testuser", "testpass")
		executor := NewWorkflowExecutor(manager)
		ctx := context.Background()

		// Test appointment booking workflow
		resp, err := executor.ExecuteIntentWorkflow(ctx, "test-session-1", IntentBook, "Book an appointment for next Monday")
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.Equal(t, "12345", resp.Data["appointment_id"])
		assert.Equal(t, "confirmed", resp.Data["status"])

		// Test transfer workflow
		resp, err = executor.ExecuteIntentWorkflow(ctx, "test-session-2", IntentTransfer, "Transfer me to a manager")
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.Equal(t, "agent-123", resp.Data["agent_id"])
	})

	t.Run("Workflow system health check", func(t *testing.T) {
		// Create mock server that returns successful health check
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/rest/workflows" {
				workflows := []map[string]interface{}{}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(workflows)
			}
		}))
		defer mockServer.Close()

		manager := NewWorkflowManager(mockServer.URL, "testuser", "testpass")
		ctx := context.Background()

		err := manager.HealthCheck(ctx)
		assert.NoError(t, err)
	})

	t.Run("Workflow system health check failure", func(t *testing.T) {
		// Create mock server that returns error
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer mockServer.Close()

		manager := NewWorkflowManager(mockServer.URL, "testuser", "testpass")
		ctx := context.Background()

		err := manager.HealthCheck(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "workflow system health check failed")
	})
}