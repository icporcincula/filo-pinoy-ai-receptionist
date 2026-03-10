// Package server contains the TEN Turn Detection implementation
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// TurnState represents the possible states from turn detection
type TurnState int

const (
	TurnStateUnfinished TurnState = iota
	TurnStateFinished
	TurnStateWait
	TurnStateUnknown
)

// String returns the string representation of the turn state
func (ts TurnState) String() string {
	switch ts {
	case TurnStateUnfinished:
		return "unfinished"
	case TurnStateFinished:
		return "finished"
	case TurnStateWait:
		return "wait"
	default:
		return "unknown"
	}
}

// TurnDetectionConfig holds the configuration for turn detection
type TurnDetectionConfig struct {
	BaseURL     string `json:"base_url"`
	APIKey      string  `json:"api_key"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"top_p"`
	ThresholdMs int     `json:"force_threshold_ms"` // <=0 means disable
}

// IsForceChatEnabled returns whether force chat is enabled
func (c *TurnDetectionConfig) IsForceChatEnabled() bool {
	return c.ThresholdMs > 0
}

// TurnDetector handles turn detection logic
type TurnDetector struct {
	config     *TurnDetectionConfig
	httpClient *http.Client
	client     *http.Client
}

// NewTurnDetector creates a new turn detector instance
func NewTurnDetector(config *TurnDetectionConfig) *TurnDetector {
	httpClient := &http.Client{
		Timeout: 5 * time.Second,
	}
	
	return &TurnDetector{
		config:     config,
		httpClient: httpClient,
		client:     httpClient,
	}
}

// Evaluate analyzes the text and determines if the speaker has finished
func (td *TurnDetector) Evaluate(ctx context.Context, text string) (TurnState, error) {
	// Remove punctuation from text (simple implementation)
	noPunctText := removePunctuation(text)
	
	// Prepare messages for the API call
	messages := []map[string]string{
		{"role": "user", "content": noPunctText},
	}

	// Create the request payload
	payload := map[string]interface{}{
		"messages":    messages,
		"model":       td.config.Model,
		"max_tokens":  1,
		"temperature": td.config.Temperature,
		"top_p":       td.config.TopP,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return TurnStateUnknown, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "POST", td.config.BaseURL+"/chat/completions", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return TurnStateUnknown, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if td.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+td.config.APIKey)
	}

	// Make the API call
	startTime := time.Now()
	resp, err := td.httpClient.Do(req)
	if err != nil {
		return TurnStateUnknown, fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return TurnStateUnknown, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return TurnStateUnknown, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	// Parse the response
	var apiResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	
	if err := json.Unmarshal(responseBody, &apiResponse); err != nil {
		return TurnStateUnknown, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResponse.Choices) == 0 || len(apiResponse.Choices[0].Message.Content) == 0 {
		return TurnStateUnknown, fmt.Errorf("no content in response")
	}

	content := strings.TrimSpace(apiResponse.Choices[0].Message.Content)
	ttfb := time.Since(startTime)
	log.Printf("Turn detection response: %s (time: %v)", content, ttfb)

	// Determine the turn state based on the response
	if strings.HasPrefix(strings.ToLower(content), "unfinished") {
		return TurnStateUnfinished, nil
	} else if strings.HasPrefix(strings.ToLower(content), "wait") {
		return TurnStateWait, nil
	} else {
		// Default to finished if it doesn't start with unfinished or wait
		return TurnStateFinished, nil
	}
}

// removePunctuation removes common punctuation from text
func removePunctuation(text string) string {
	var result strings.Builder
	for _, r := range text {
		switch r {
	case '.', '!', '?', ',', ';', ':', '"', '\'', '(', ')', '[', ']', '{', '}':
			continue
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// Close cleans up resources
func (td *TurnDetector) Close() error {
	// In this implementation, we don't have persistent connections to close
	// but in a real implementation you might close HTTP clients or other resources
	return nil
}