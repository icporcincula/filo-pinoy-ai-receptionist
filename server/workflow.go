package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// WorkflowManager handles n8n workflow integration
type WorkflowManager struct {
	baseURL     string
	apiKey      string
	httpClient  *http.Client
	credentials *WorkflowCredentials
}

// WorkflowCredentials stores authentication for n8n
type WorkflowCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// WorkflowTrigger represents a workflow trigger request
type WorkflowTrigger struct {
	WorkflowID string                 `json:"workflow_id"`
	Input      map[string]interface{} `json:"input"`
}

// WorkflowResponse represents the response from n8n
type WorkflowResponse struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// NewWorkflowManager creates a new workflow manager
func NewWorkflowManager(baseURL, username, password string) *WorkflowManager {
	return &WorkflowManager{
		baseURL: baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		credentials: &WorkflowCredentials{
			Username: username,
			Password: password,
		},
	}
}

// TriggerWorkflow triggers a specific workflow in n8n
func (wm *WorkflowManager) TriggerWorkflow(ctx context.Context, workflowID string, input map[string]interface{}) (*WorkflowResponse, error) {
	trigger := WorkflowTrigger{
		WorkflowID: workflowID,
		Input:      input,
	}
	
	reqBody, err := json.Marshal(trigger)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal workflow trigger: %w", err)
	}
	
	url := fmt.Sprintf("%s/webhook/%s", wm.baseURL, workflowID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	// Set basic authentication
	req.SetBasicAuth(wm.credentials.Username, wm.credentials.Password)
	
	resp, err := wm.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to trigger workflow: %w", err)
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow response: %w", err)
	}
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("workflow trigger failed with status %d: %s", resp.StatusCode, string(respBody))
	}
	
	var workflowResp WorkflowResponse
	err = json.Unmarshal(respBody, &workflowResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal workflow response: %w", err)
	}
	
	return &workflowResp, nil
}

// ListWorkflows retrieves available workflows from n8n
func (wm *WorkflowManager) ListWorkflows(ctx context.Context) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/rest/workflows", wm.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list workflows request: %w", err)
	}
	
	req.SetBasicAuth(wm.credentials.Username, wm.credentials.Password)
	
	resp, err := wm.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflows response: %w", err)
	}
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("list workflows failed with status %d: %s", resp.StatusCode, string(respBody))
	}
	
	var workflows []map[string]interface{}
	err = json.Unmarshal(respBody, &workflows)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal workflows response: %w", err)
	}
	
	return workflows, nil
}

// GetWorkflow retrieves a specific workflow by ID
func (wm *WorkflowManager) GetWorkflow(ctx context.Context, workflowID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/rest/workflows/%s", wm.baseURL, workflowID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get workflow request: %w", err)
	}
	
	req.SetBasicAuth(wm.credentials.Username, wm.credentials.Password)
	
	resp, err := wm.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow response: %w", err)
	}
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("get workflow failed with status %d: %s", resp.StatusCode, string(respBody))
	}
	
	var workflow map[string]interface{}
	err = json.Unmarshal(respBody, &workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal workflow response: %w", err)
	}
	
	return workflow, nil
}

// WorkflowRegistry manages workflow mappings and routing
type WorkflowRegistry struct {
	workflows map[string]string // intent -> workflow ID mapping
	manager   *WorkflowManager
}

// NewWorkflowRegistry creates a new workflow registry
func NewWorkflowRegistry(manager *WorkflowManager) *WorkflowRegistry {
	return &WorkflowRegistry{
		workflows: map[string]string{
			"appointment_booking": "appointment-booking-workflow",
			"transfer_to_agent":   "transfer-to-agent-workflow", 
			"knowledge_search":    "knowledge-search-workflow",
		},
		manager: manager,
	}
}

// RegisterWorkflow registers a new workflow mapping
func (wr *WorkflowRegistry) RegisterWorkflow(intent string, workflowID string) {
	wr.workflows[intent] = workflowID
}

// GetWorkflowID retrieves the workflow ID for an intent
func (wr *WorkflowRegistry) GetWorkflowID(intent string) string {
	if workflowID, exists := wr.workflows[intent]; exists {
		return workflowID
	}
	return ""
}

// ExecuteWorkflow executes a workflow for a given intent
func (wr *WorkflowRegistry) ExecuteWorkflow(ctx context.Context, intent string, input map[string]interface{}) (*WorkflowResponse, error) {
	workflowID := wr.GetWorkflowID(intent)
	if workflowID == "" {
		return nil, fmt.Errorf("no workflow registered for intent: %s", intent)
	}
	
	return wr.manager.TriggerWorkflow(ctx, workflowID, input)
}

// WorkflowLogger logs workflow executions
type WorkflowLogger struct{}

// LogWorkflowExecution logs workflow execution details
func (wl *WorkflowLogger) LogWorkflowExecution(sessionID, intent, workflowID string, input, output map[string]interface{}, success bool) {
	log.Printf("[Workflow] Session: %s, Intent: %s, Workflow: %s, Success: %v, Input: %v, Output: %v",
		sessionID, intent, workflowID, success, input, output)
}

// WorkflowError represents a workflow execution error
type WorkflowError struct {
	WorkflowID string `json:"workflow_id"`
	Error      string `json:"error"`
	Timestamp  int64  `json:"timestamp"`
}

// WorkflowMonitor monitors workflow executions and errors
type WorkflowMonitor struct {
	errors []WorkflowError
	logger *WorkflowLogger
}

// NewWorkflowMonitor creates a new workflow monitor
func NewWorkflowMonitor() *WorkflowMonitor {
	return &WorkflowMonitor{
		errors: make([]WorkflowError, 0),
		logger: &WorkflowLogger{},
	}
}

// RecordError records a workflow execution error
func (wm *WorkflowMonitor) RecordError(workflowID, errorStr string) {
	wm.errors = append(wm.errors, WorkflowError{
		WorkflowID: workflowID,
		Error:      errorStr,
		Timestamp:  time.Now().Unix(),
	})
}

// GetErrors returns all recorded errors
func (wm *WorkflowMonitor) GetErrors() []WorkflowError {
	return wm.errors
}

// ClearErrors clears all recorded errors
func (wm *WorkflowMonitor) ClearErrors() {
	wm.errors = make([]WorkflowError, 0)
}

// WorkflowMetrics tracks workflow execution metrics
type WorkflowMetrics struct {
	TotalExecutions int                    `json:"total_executions"`
	SuccessRate     float64               `json:"success_rate"`
	AverageDuration time.Duration         `json:"average_duration"`
	TopWorkflows    map[string]int        `json:"top_workflows"`
	ErrorCount      int                   `json:"error_count"`
}

// GetMetrics returns workflow execution metrics
func (wm *WorkflowMonitor) GetMetrics() WorkflowMetrics {
	// In a real implementation, this would aggregate from a database
	return WorkflowMetrics{
		TotalExecutions: 0,
		SuccessRate:     0.0,
		AverageDuration: 0,
		TopWorkflows:    make(map[string]int),
		ErrorCount:      len(wm.errors),
	}
}

// WorkflowExecutor provides high-level workflow execution
type WorkflowExecutor struct {
	registry *WorkflowRegistry
	monitor  *WorkflowMonitor
	logger   *WorkflowLogger
}

// NewWorkflowExecutor creates a new workflow executor
func NewWorkflowExecutor(manager *WorkflowManager) *WorkflowExecutor {
	return &WorkflowExecutor{
		registry: NewWorkflowRegistry(manager),
		monitor:  NewWorkflowMonitor(),
		logger:   &WorkflowLogger{},
	}
}

// ExecuteIntentWorkflow executes the appropriate workflow for an intent
func (we *WorkflowExecutor) ExecuteIntentWorkflow(ctx context.Context, sessionID string, intent Intent, userInput string) (*WorkflowResponse, error) {
	// Prepare input for the workflow
	input := map[string]interface{}{
		"session_id": sessionID,
		"user_input": userInput,
		"intent":     string(intent),
		"timestamp":  time.Now().Unix(),
		"business_name": getBusinessName(),
	}
	
	// Execute the workflow
	workflowID := we.registry.GetWorkflowID(string(intent))
	if workflowID == "" {
		return nil, fmt.Errorf("no workflow found for intent: %s", intent)
	}
	
	startTime := time.Now()
	resp, err := we.registry.ExecuteWorkflow(ctx, string(intent), input)
	duration := time.Since(startTime)
	
	// Log the execution
	success := err == nil && resp != nil && resp.Success
	we.logger.LogWorkflowExecution(sessionID, string(intent), workflowID, input, resp.Data, success)
	
	if err != nil {
		we.monitor.RecordError(workflowID, err.Error())
		return nil, err
	}
	
	if !resp.Success {
		we.monitor.RecordError(workflowID, resp.Error)
		return nil, fmt.Errorf("workflow execution failed: %s", resp.Error)
	}
	
	log.Printf("[Workflow] Execution completed in %v for intent: %s", duration, intent)
	return resp, nil
}

// HealthCheck checks if the workflow system is healthy
func (wm *WorkflowManager) HealthCheck(ctx context.Context) error {
	// Try to list workflows to check connectivity
	_, err := wm.ListWorkflows(ctx)
	if err != nil {
		return fmt.Errorf("workflow system health check failed: %w", err)
	}
	return nil
}