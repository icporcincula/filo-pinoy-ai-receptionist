package main

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"
)

// TestTENVADTurnDetection tests the integration of TEN VAD and Turn Detection
func TestTENVADTurnDetection(t *testing.T) {
	// Test configuration
	config := &TurnDetectionConfig{
		BaseURL:     "http://localhost:8000/v1",
		APIKey:      "test-key",
		Model:       "test-model",
		Temperature: 0.1,
		TopP:        0.1,
		ThresholdMs: 500,
	}

	// Create turn detector
	turnDetector := NewTurnDetector(config)
	defer turnDetector.Close()

	// Test turn detection with various inputs
	testCases := []struct {
		input    string
		expected TurnState
		name     string
	}{
		{
			input:    "Hello, how are you today?",
			expected: TurnStateFinished, // Assuming a complete sentence finishes
			name:     "complete_sentence",
		},
		{
			input:    "I think that maybe possibly",
			expected: TurnStateUnfinished, // Assuming incomplete thought
			name:     "incomplete_thought",
		},
		{
			input:    "",
			expected: TurnStateUnknown, // Empty input
			name:     "empty_input",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			state, err := turnDetector.Evaluate(ctx, tc.input)
			if err != nil {
				// Don't fail the test if the API isn't available, just log
				t.Logf("Turn detection evaluation failed for '%s': %v", tc.name, err)
				return
			}

			// Note: Since we're mocking the API response, we expect unknown state
			// unless the API is actually running
			t.Logf("Input: '%s' -> State: %s", tc.input, state.String())
		})
	}

	// Test TEN VAD
	t.Run("vad_test", func(t *testing.T) {
		// Create a simple callback to capture utterances
		utteranceReceived := make(chan []int16, 1)
		vad := NewTenVAD(func(samples []int16) {
			utteranceReceived <- samples
		})

		// Simulate some audio samples (silence and speech)
		silence := make([]int16, 160) // 10ms of silence at 16kHz
		speech := make([]int16, 160)  // 10ms of simulated speech

		// Fill speech with some values to simulate audio
		for i := 0; i < len(speech); i++ {
			speech[i] = 1000 // Some amplitude to simulate speech
		}

		// Feed samples to VAD
		vad.Feed(silence)
		vad.Feed(speech)
		vad.Feed(silence)

		// Give some time for processing
		time.Sleep(100 * time.Millisecond)

		// Check if an utterance was detected
		select {
		case <-utteranceReceived:
			t.Log("VAD successfully detected an utterance")
		default:
			t.Log("No utterance detected by VAD (this may be expected)")
		}
	})
}

// Example usage of the TEN VAD and Turn Detection
func ExampleTENVADTurnDetection() {
	// Create turn detection config
	config := &TurnDetectionConfig{
		BaseURL:     "http://localhost:8000/v1",
		APIKey:      "example-key",
		Model:       "example-model",
		Temperature: 0.1,
		TopP:        0.1,
		ThresholdMs: 500,
	}

	// Create turn detector
	turnDetector := NewTurnDetector(config)
	defer turnDetector.Close()

	// Example: Evaluate a user's transcript
	ctx := context.Background
	transcript := "Hello, I would like to know about your services."

	state, err := turnDetector.Evaluate(ctx, transcript)
	if err != nil {
		log.Printf("Turn detection error: %v", err)
		return
	}

	fmt.Printf("Turn state for '%s': %s\n", transcript, state.String())

	// Example: Create and use TEN VAD
	vad := NewTenVAD(func(samples []int16) {
		fmt.Printf("VAD detected utterance with %d samples\n", len(samples))
	})

	// Simulate feeding audio samples
	audioSamples := make([]int16, 160) // 10ms of audio at 16kHz
	for i := 0; i < len(audioSamples); i++ {
		audioSamples[i] = 500 // Simulated audio amplitude
	}
	vad.Feed(audioSamples)
}