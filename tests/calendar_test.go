package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock HTTP client for testing
type MockCalendarHTTPClient struct {
	mock.Mock
}

func (m *MockCalendarHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

// Test CalendarManager
func TestCalendarManager(t *testing.T) {
	t.Run("NewCalendarManager", func(t *testing.T) {
		credentials := &CalendarCredentials{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RefreshToken: "test-refresh-token",
			AccessToken:  "test-access-token",
		}
		
		manager := NewCalendarManager("test-calendar-id", credentials)
		assert.NotNil(t, manager)
		assert.Equal(t, "test-calendar-id", manager.calendarID)
		assert.Equal(t, credentials, manager.credentials)
		assert.NotNil(t, manager.httpClient)
	})

	t.Run("GetAccessToken", func(t *testing.T) {
		credentials := &CalendarCredentials{
			AccessToken: "test-access-token",
		}
		
		manager := NewCalendarManager("test-calendar-id", credentials)
		ctx := context.Background()
		
		token, err := manager.GetAccessToken(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "test-access-token", token)
	})

	t.Run("ListEvents", func(t *testing.T) {
		// Create mock server
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Contains(t, r.URL.Path, "/calendars/test-calendar-id/events")
			assert.Contains(t, r.URL.RawQuery, "timeMin=")
			assert.Contains(t, r.URL.RawQuery, "timeMax=")
			
			// Verify authorization header
			authHeader := r.Header.Get("Authorization")
			assert.Contains(t, authHeader, "Bearer ")
			
			// Return mock events
			eventsResponse := struct {
				Items []CalendarEvent `json:"items"`
			}{
				Items: []CalendarEvent{
					{
						Summary: "Test Meeting",
						Description: "Test meeting description",
						Start: EventDateTime{
							DateTime: "2024-01-15T10:00:00+08:00",
							TimeZone: "Asia/Manila",
						},
						End: EventDateTime{
							DateTime: "2024-01-15T11:00:00+08:00",
							TimeZone: "Asia/Manila",
						},
						Attendees: []EventAttendee{
							{Email: "test@example.com"},
						},
						Location: "Virtual Meeting",
					},
				},
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(eventsResponse)
		}))
		defer mockServer.Close()

		credentials := &CalendarCredentials{
			AccessToken: "test-access-token",
		}
		manager := NewCalendarManager(mockServer.URL, credentials)
		ctx := context.Background()

		startTime := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
		endTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

		events, err := manager.ListEvents(ctx, startTime, endTime)
		assert.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, "Test Meeting", events[0].Summary)
		assert.Equal(t, "Test meeting description", events[0].Description)
		assert.Equal(t, "Virtual Meeting", events[0].Location)
		assert.Len(t, events[0].Attendees, 1)
		assert.Equal(t, "test@example.com", events[0].Attendees[0].Email)
	})

	t.Run("CreateEvent", func(t *testing.T) {
		// Create mock server
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Contains(t, r.URL.Path, "/calendars/test-calendar-id/events")
			
			// Verify request body
			body, _ := io.ReadAll(r.Body)
			var event CalendarEvent
			err := json.Unmarshal(body, &event)
			assert.NoError(t, err)
			assert.Equal(t, "Test Appointment", event.Summary)
			assert.Equal(t, "Test appointment description", event.Description)
			
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
		}))
		defer mockServer.Close()

		credentials := &CalendarCredentials{
			AccessToken: "test-access-token",
		}
		manager := NewCalendarManager(mockServer.URL, credentials)
		ctx := context.Background()

		event := &CalendarEvent{
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

		createdEvent, err := manager.CreateEvent(ctx, event)
		assert.NoError(t, err)
		assert.NotNil(t, createdEvent)
		assert.Equal(t, "Test Appointment", createdEvent.Summary)
		assert.Equal(t, "Test appointment description", createdEvent.Description)
	})

	t.Run("CheckAvailability", func(t *testing.T) {
		// Create mock server that returns no conflicting events
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			eventsResponse := struct {
				Items []CalendarEvent `json:"items"`
			}{Items: []CalendarEvent{}}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(eventsResponse)
		}))
		defer mockServer.Close()

		credentials := &CalendarCredentials{
			AccessToken: "test-access-token",
		}
		manager := NewCalendarManager(mockServer.URL, credentials)
		ctx := context.Background()

		startTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		endTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

		available, err := manager.CheckAvailability(ctx, startTime, endTime)
		assert.NoError(t, err)
		assert.True(t, available)
	})

	t.Run("CheckAvailability with conflict", func(t *testing.T) {
		// Create mock server that returns conflicting event
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			eventsResponse := struct {
				Items []CalendarEvent `json:"items"`
			}{
				Items: []CalendarEvent{
					{
						Summary: "Existing Meeting",
						Start: EventDateTime{
							DateTime: "2024-01-15T09:30:00+08:00",
							TimeZone: "Asia/Manila",
						},
						End: EventDateTime{
							DateTime: "2024-01-15T10:30:00+08:00",
							TimeZone: "Asia/Manila",
						},
					},
				},
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(eventsResponse)
		}))
		defer mockServer.Close()

		credentials := &CalendarCredentials{
			AccessToken: "test-access-token",
		}
		manager := NewCalendarManager(mockServer.URL, credentials)
		ctx := context.Background()

		startTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		endTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

		available, err := manager.CheckAvailability(ctx, startTime, endTime)
		assert.NoError(t, err)
		assert.False(t, available)
	})
}

// Test CalendarExecutor
func TestCalendarExecutor(t *testing.T) {
	t.Run("NewCalendarExecutor", func(t *testing.T) {
		credentials := &CalendarCredentials{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RefreshToken: "test-refresh-token",
			AccessToken:  "test-access-token",
		}
		
		executor := NewCalendarExecutor("test-calendar-id", credentials)
		assert.NotNil(t, executor)
		assert.NotNil(t, executor.manager)
		assert.NotNil(t, executor.monitor)
		assert.NotNil(t, executor.logger)
		assert.Equal(t, "test-calendar-id", executor.manager.calendarID)
	})

	t.Run("BookAppointment", func(t *testing.T) {
		// Create mock server
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		defer mockServer.Close()

		credentials := &CalendarCredentials{
			AccessToken: "test-access-token",
		}
		executor := NewCalendarExecutor(mockServer.URL, credentials)
		ctx := context.Background()

		startTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		endTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

		createdEvent, err := executor.BookAppointment(ctx, "test-session", 
			"Test Appointment", "Test appointment description", 
			startTime, endTime, []string{"test@example.com"})
		
		assert.NoError(t, err)
		assert.NotNil(t, createdEvent)
		assert.Equal(t, "Test Appointment", createdEvent.Summary)
		assert.Equal(t, "Test appointment description", createdEvent.Description)
	})

	t.Run("BookAppointment with conflict", func(t *testing.T) {
		// Create mock server that returns conflicting event
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" {
				// Return conflicting event
				eventsResponse := struct {
					Items []CalendarEvent `json:"items"`
				}{
					Items: []CalendarEvent{
						{
							Summary: "Existing Meeting",
							Start: EventDateTime{
								DateTime: "2024-01-15T09:30:00+08:00",
								TimeZone: "Asia/Manila",
							},
							End: EventDateTime{
								DateTime: "2024-01-15T10:30:00+08:00",
								TimeZone: "Asia/Manila",
							},
						},
					},
				}
				
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(eventsResponse)
			}
		}))
		defer mockServer.Close()

		credentials := &CalendarCredentials{
			AccessToken: "test-access-token",
		}
		executor := NewCalendarExecutor(mockServer.URL, credentials)
		ctx := context.Background()

		startTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		endTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

		createdEvent, err := executor.BookAppointment(ctx, "test-session", 
			"Test Appointment", "Test appointment description", 
			startTime, endTime, []string{"test@example.com"})
		
		assert.Error(t, err)
		assert.Nil(t, createdEvent)
		assert.Contains(t, err.Error(), "time slot is not available")
	})

	t.Run("GetAvailableSlots", func(t *testing.T) {
		// Create mock server that returns some events
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			eventsResponse := struct {
				Items []CalendarEvent `json:"items"`
			}{
				Items: []CalendarEvent{
					{
						Summary: "Morning Meeting",
						Start: EventDateTime{
							DateTime: "2024-01-15T09:30:00+08:00",
							TimeZone: "Asia/Manila",
						},
						End: EventDateTime{
							DateTime: "2024-01-15T10:30:00+08:00",
							TimeZone: "Asia/Manila",
						},
					},
				},
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(eventsResponse)
		}))
		defer mockServer.Close()

		credentials := &CalendarCredentials{
			AccessToken: "test-access-token",
		}
		executor := NewCalendarExecutor(mockServer.URL, credentials)
		ctx := context.Background()

		date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		duration := 30 * time.Minute

		slots, err := executor.GetAvailableSlots(ctx, "test-session", date, duration)
		assert.NoError(t, err)
		assert.NotNil(t, slots)
		assert.Greater(t, len(slots), 0)
		
		// Verify that slots don't overlap with the existing meeting
		for _, slot := range slots {
			slotStart, _ := time.Parse(time.RFC3339, slot.Start)
			slotEnd, _ := time.Parse(time.RFC3339, slot.End)
			
			// Should not overlap with 09:30-10:30
			existingStart := time.Date(2024, 1, 15, 9, 30, 0, 0, time.UTC)
			existingEnd := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
			
			overlap := (slotStart.Before(existingEnd) && slotEnd.After(existingStart))
			assert.False(t, overlap, "Slot %s-%s should not overlap with existing meeting", slot.Start, slot.End)
		}
	})
}

// Test CalendarLogger
func TestCalendarLogger(t *testing.T) {
	t.Run("LogCalendarOperation", func(t *testing.T) {
		logger := &CalendarLogger{}
		
		details := map[string]interface{}{
			"event_id": "12345",
			"summary":  "Test Appointment",
		}
		
		// This should not panic and should log the operation
		logger.LogCalendarOperation("book_appointment", "test-session", true, details)
	})
}

// Test CalendarMonitor
func TestCalendarMonitor(t *testing.T) {
	t.Run("NewCalendarMonitor", func(t *testing.T) {
		monitor := NewCalendarMonitor()
		assert.NotNil(t, monitor)
		assert.NotNil(t, monitor.errors)
		assert.NotNil(t, monitor.logger)
		assert.Empty(t, monitor.errors)
	})

	t.Run("RecordError", func(t *testing.T) {
		monitor := NewCalendarMonitor()
		
		monitor.RecordError("test_operation", "Test error message")
		errors := monitor.GetErrors()
		assert.Len(t, errors, 1)
		assert.Equal(t, "test_operation", errors[0].Operation)
		assert.Equal(t, "Test error message", errors[0].Error)
		assert.NotZero(t, errors[0].Timestamp)
	})

	t.Run("ClearErrors", func(t *testing.T) {
		monitor := NewCalendarMonitor()
		
		monitor.RecordError("test_operation", "Test error message")
		assert.Len(t, monitor.GetErrors(), 1)
		
		monitor.ClearErrors()
		assert.Empty(t, monitor.GetErrors())
	})

	t.Run("GetMetrics", func(t *testing.T) {
		monitor := NewCalendarMonitor()
		metrics := monitor.GetMetrics()
		
		assert.Equal(t, 0, metrics.TotalOperations)
		assert.Equal(t, 0.0, metrics.SuccessRate)
		assert.Equal(t, time.Duration(0), metrics.AverageDuration)
		assert.Equal(t, 0, metrics.BookedAppointments)
		assert.Equal(t, 0, metrics.CancelledAppointments)
	})
}

// Test CalendarMetrics
func TestCalendarMetrics(t *testing.T) {
	t.Run("GetMetrics", func(t *testing.T) {
		monitor := NewCalendarMonitor()
		metrics := monitor.GetMetrics()
		
		assert.Equal(t, 0, metrics.TotalOperations)
		assert.Equal(t, 0.0, metrics.SuccessRate)
		assert.Equal(t, time.Duration(0), metrics.AverageDuration)
		assert.Equal(t, 0, metrics.BookedAppointments)
		assert.Equal(t, 0, metrics.CancelledAppointments)
	})
}

// Test CalendarError
func TestCalendarError(t *testing.T) {
	t.Run("CalendarError creation", func(t *testing.T) {
		timestamp := time.Now().Unix()
		error := CalendarError{
			Operation: "test_operation",
			Error:     "Test error",
			Timestamp: timestamp,
		}
		
		assert.Equal(t, "test_operation", error.Operation)
		assert.Equal(t, "Test error", error.Error)
		assert.Equal(t, timestamp, error.Timestamp)
	})
}

// Benchmark tests
func BenchmarkCalendarManager_ListEvents(b *testing.B) {
	// Create mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		eventsResponse := struct {
			Items []CalendarEvent `json:"items"`
		}{Items: []CalendarEvent{}}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(eventsResponse)
	}))
	defer mockServer.Close()

	credentials := &CalendarCredentials{
		AccessToken: "test-access-token",
	}
	manager := NewCalendarManager(mockServer.URL, credentials)
	ctx := context.Background()
	startTime := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
	endTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.ListEvents(ctx, startTime, endTime)
	}
}

func BenchmarkCalendarManager_CheckAvailability(b *testing.B) {
	// Create mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		eventsResponse := struct {
			Items []CalendarEvent `json:"items"`
		}{Items: []CalendarEvent{}}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(eventsResponse)
	}))
	defer mockServer.Close()

	credentials := &CalendarCredentials{
		AccessToken: "test-access-token",
	}
	manager := NewCalendarManager(mockServer.URL, credentials)
	ctx := context.Background()
	startTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.CheckAvailability(ctx, startTime, endTime)
	}
}

// Integration test for calendar system
func TestCalendarSystemIntegration(t *testing.T) {
	t.Run("End-to-end calendar operations", func(t *testing.T) {
		// Create mock server that simulates a full calendar workflow
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" {
				// Return some existing events
				eventsResponse := struct {
					Items []CalendarEvent `json:"items"`
				}{
					Items: []CalendarEvent{
						{
							Summary: "Morning Meeting",
							Start: EventDateTime{
								DateTime: "2024-01-15T09:30:00+08:00",
								TimeZone: "Asia/Manila",
							},
							End: EventDateTime{
								DateTime: "2024-01-15T10:30:00+08:00",
								TimeZone: "Asia/Manila",
							},
						},
					},
				}
				
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(eventsResponse)
			} else if r.Method == "POST" {
				// Return created event
				createdEvent := CalendarEvent{
					Summary:     "Test Appointment",
					Description: "Test appointment description",
					Start: EventDateTime{
						DateTime: "2024-01-15T11:00:00+08:00",
						TimeZone: "Asia/Manila",
					},
					End: EventDateTime{
						DateTime: "2024-01-15T11:30:00+08:00",
						TimeZone: "Asia/Manila",
					},
					Location: "Virtual Meeting",
				}
				
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(createdEvent)
			}
		}))
		defer mockServer.Close()

		credentials := &CalendarCredentials{
			AccessToken: "test-access-token",
		}
		executor := NewCalendarExecutor(mockServer.URL, credentials)
		ctx := context.Background()

		// Test getting available slots
		date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		duration := 30 * time.Minute

		slots, err := executor.GetAvailableSlots(ctx, "test-session-1", date, duration)
		assert.NoError(t, err)
		assert.NotNil(t, slots)
		assert.Greater(t, len(slots), 0)

		// Test booking an appointment in an available slot
		if len(slots) > 0 {
			slotStart, _ := time.Parse(time.RFC3339, slots[0].Start)
			slotEnd, _ := time.Parse(time.RFC3339, slots[0].End)

			createdEvent, err := executor.BookAppointment(ctx, "test-session-2", 
				"Test Appointment", "Test appointment description", 
				slotStart, slotEnd, []string{"test@example.com"})
			
			assert.NoError(t, err)
			assert.NotNil(t, createdEvent)
			assert.Equal(t, "Test Appointment", createdEvent.Summary)
		}
	})

	t.Run("Calendar system health check", func(t *testing.T) {
		// Create mock server that returns successful health check
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/calendars/test-calendar-id/events" {
				eventsResponse := struct {
					Items []CalendarEvent `json:"items"`
				}{Items: []CalendarEvent{}}
				
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(eventsResponse)
			}
		}))
		defer mockServer.Close()

		credentials := &CalendarCredentials{
			AccessToken: "test-access-token",
		}
		manager := NewCalendarManager(mockServer.URL, credentials)
		ctx := context.Background()

		err := manager.HealthCheck(ctx)
		assert.NoError(t, err)
	})

	t.Run("Calendar system health check failure", func(t *testing.T) {
		// Create mock server that returns error
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer mockServer.Close()

		credentials := &CalendarCredentials{
			AccessToken: "test-access-token",
		}
		manager := NewCalendarManager(mockServer.URL, credentials)
		ctx := context.Background()

		err := manager.HealthCheck(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "calendar system health check failed")
	})
}