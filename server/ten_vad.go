// Package server contains the TEN VAD implementation
package main

import (
	"math"
	"sync"
	"time"
)

// TenVAD represents the TEN Voice Activity Detection implementation
type TenVAD struct {
	mu           sync.Mutex
	buf          []int16
	silenceSamps int
	speechSamps  int
	active       bool
	mode         int
	frameCounter int
	speechCount  int
	noiseCount   int
	onUtterance  func([]int16)
}

// NewTenVAD creates a new TEN VAD instance
func NewTenVAD(onUtterance func([]int16)) *TenVAD {
	return &TenVAD{
		onUtterance: onUtterance,
		mode:        2, // Default to mode 2 (balanced)
	}
}

// Feed processes audio samples and detects voice activity
func (v *TenVAD) Feed(samples []int16) {
	isSpeech := v.process(samples)
	
	v.mu.Lock()
	defer v.mu.Unlock()

	// Constants for timing (similar to original VAD)
	silenceThresh := 1200 * vadSampleRate / 1000 // 1200ms of silence
	minSpeech := 10 * vadSampleRate / 1000     // 100ms minimum speech

	if isSpeech {
		v.silenceSamps = 0
		v.speechSamps += len(samples)
		v.active = true
		v.buf = append(v.buf, samples...)
	} else if v.active {
		v.silenceSamps += len(samples)
		v.buf = append(v.buf, samples...)
		if v.silenceSamps >= silenceThresh {
			if v.speechSamps >= minSpeech {
				out := make([]int16, len(v.buf))
				copy(out, v.buf)
				go v.onUtterance(out)
			} else {
				// Utterance too short
			}
			v.buf = v.buf[:0]
			v.speechSamps = 0
			v.silenceSamps = 0
			v.active = false
		}
	}
}

// process implements the core VAD logic similar to the C implementation
func (v *TenVAD) process(audioFrame []int16) bool {
	// Calculate RMS (Root Mean Square) energy
	energy := calculateRMS(audioFrame)

	// Energy threshold based on mode (more aggressive = higher threshold)
	threshold := 500.0 + float64(v.mode)*300.0

	isSpeech := 0
	if energy > threshold {
		isSpeech = 1
	}

	// Apply smoothing based on history
	if isSpeech == 1 {
		v.speechCount++
		v.noiseCount = 0
	} else {
		v.noiseCount++
		v.speechCount = 0
	}

	// Require multiple consecutive frames for state change
	hysteresis := (3 - v.mode) // Less hysteresis for more aggressive modes

	if v.speechCount > hysteresis {
		return true
	} else if v.noiseCount > hysteresis {
		return false
	}

	// Return previous state if in transition
	return v.speechCount > 0
}

// calculateRMS calculates the Root Mean Square energy of audio samples
func calculateRMS(samples []int16) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for _, v := range samples {
		f := float64(v)
		sum += f * f
	}
	return math.Sqrt(sum / float64(len(samples)))
}

// SetMode sets the aggressiveness mode (0-3)
func (v *TenVAD) SetMode(mode int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	if mode < 0 || mode > 3 {
		return
	}
	v.mode = mode
}