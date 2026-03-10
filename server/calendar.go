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

// CalendarManager handles Google Calendar integration
type CalendarManager struct {
	calendarID string
	httpClient *http.Client
	credentials *CalendarCredentials
}

// CalendarCredentials stores Google Calendar API credentials
type CalendarCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

// CalendarEvent represents a calendar event
type CalendarEvent struct {
	Summary     string            `json:"summary"`
	Description string            `json:"description"`
	Start       EventDateTime     `json:"start"`
	End         EventDateTime     `json:"end"`
	Attendees   []EventAttendee   `json:"attendees,omitempty"`
	Location    string            `json:"location,omitempty"`
}

// EventDateTime represents event start/end times
type EventDateTime struct {
	DateTime string `json:"dateTime"`
	TimeZone string `json:"timeZone"`
}

// EventAttendee represents an event attendee
type EventAttendee struct {
	Email string `json:"email"`
}

// CalendarAvailability represents available time slots
type CalendarAvailability struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// NewCalendarManager creates a new calendar manager
func NewCalendarManager(calendarID string, credentials *CalendarCredentials) *CalendarManager {
	return &CalendarManager{
		calendarID:  calendarID,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		credentials: credentials,
	}
}

// GetAccessToken refreshes the access token if needed
func (cm *CalendarManager) GetAccessToken(ctx context.Context) (string, error) {
	if cm.credentials.AccessToken != "" {
		return cm.credentials.AccessToken, nil
	}
	
	// In a real implementation, you would refresh the token using the refresh token
	// For now, return the existing token or error
	if cm.credentials.RefreshToken == "" {
		return "", fmt.Errorf("no access token or refresh token available")
	}
	
	// This is a placeholder - in production, implement proper OAuth2 token refresh
	return cm.credentials.AccessToken, nil
}

// ListEvents retrieves events from the calendar
func (cm *CalendarManager) ListEvents(ctx context.Context, timeMin, timeMax time.Time) ([]CalendarEvent, error) {
	accessToken, err := cm.GetAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}
	
	url := fmt.Sprintf("https://www.googleapis.com/calendar/v3/calendars/%s/events", cm.calendarID)
	url += fmt.Sprintf("?timeMin=%s&timeMax=%s&singleEvents=true&orderBy=startTime",
		timeMin.Format(time.RFC3339), timeMax.Format(time.RFC3339))
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list events request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := cm.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read events response: %w", err)
	}
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("list events failed with status %d: %s", resp.StatusCode, string(respBody))
	}
	
	var eventsResponse struct {
		Items []CalendarEvent `json:"items"`
	}
	err = json.Unmarshal(respBody, &eventsResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal events response: %w", err)
	}
	
	return eventsResponse.Items, nil
}

// CreateEvent creates a new calendar event
func (cm *CalendarManager) CreateEvent(ctx context.Context, event *CalendarEvent) (*CalendarEvent, error) {
	accessToken, err := cm.GetAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}
	
	reqBody, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}
	
	url := fmt.Sprintf("https://www.googleapis.com/calendar/v3/calendars/%s/events", cm.calendarID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create create event request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := cm.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read create event response: %w", err)
	}
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("create event failed with status %d: %s", resp.StatusCode, string(respBody))
	}
	
	var createdEvent CalendarEvent
	err = json.Unmarshal(respBody, &createdEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal created event response: %w", err)
	}
	
	return &createdEvent, nil
}

// CheckAvailability checks if a time slot is available
func (cm *CalendarManager) CheckAvailability(ctx context.Context, start, end time.Time) (bool, error) {
	events, err := cm.ListEvents(ctx, start.Add(-1*time.Hour), end.Add(1*time.Hour))
	if err != nil {
		return false, fmt.Errorf("failed to check availability: %w", err)
	}
	
	for _, event := range events {
		eventStart, err := time.Parse(time.RFC3339, event.Start.DateTime)
		if err != nil {
			continue
		}
		eventEnd, err := time.Parse(time.RFC3339, event.End.DateTime)
		if err != nil {
			continue
		}
		
		// Check for overlap
		if (start.Before(eventEnd) && end.After(eventStart)) ||
		   (eventStart.Before(end) && eventEnd.After(start)) {
			return false, nil
		}
	}
	
	return true, nil
}

// FindAvailableSlots finds available time slots for appointments
func (cm *CalendarManager) FindAvailableSlots(ctx context.Context, date time.Time, duration time.Duration) ([]CalendarAvailability, error) {
	// Start from business hours (9 AM)
	startTime := time.Date(date.Year(), date.Month(), date.Day(), 9, 0, 0, 0, date.Location())
	endTime := startTime.Add(9 * time.Hour) // End at 6 PM
	
	// Get existing events for the day
	events, err := cm.ListEvents(ctx, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get events for availability check: %w", err)
	}
	
	var availableSlots []CalendarAvailability
	currentTime := startTime
	
	for currentTime.Add(duration).Before(endTime) {
		slotEnd := currentTime.Add(duration)
		
		// Check if this slot conflicts with any existing events
		conflict := false
		for _, event := range events {
			eventStart, err := time.Parse(time.RFC3339, event.Start.DateTime)
			if err != nil {
				continue
			}
			eventEnd, err := time.Parse(time.RFC3339, event.End.DateTime)
			if err != nil {
				continue
			}
			
			// Check for overlap
			if (currentTime.Before(eventEnd) && slotEnd.After(eventStart)) ||
			   (eventStart.Before(slotEnd) && eventEnd.After(currentTime)) {
				conflict = true
				currentTime = eventEnd
				break
			}
		}
		
		if !conflict {
			availableSlots = append(availableSlots, CalendarAvailability{
				Start: currentTime.Format(time.RFC3339),
				End:   slotEnd.Format(time.RFC3339),
			})
			currentTime = currentTime.Add(30 * time.Minute) // Check every 30 minutes
		}
	}
	
	return availableSlots, nil
}

// CalendarLogger logs calendar operations
type CalendarLogger struct{}

// LogCalendarOperation logs calendar operations
func (cl *CalendarLogger) LogCalendarOperation(operation, sessionID string, success bool, details map[string]interface{}) {
	log.Printf("[Calendar] Operation: %s, Session: %s, Success: %v, Details: %v",
		operation, sessionID, success, details)
}

// CalendarError represents a calendar operation error
type CalendarError struct {
	Operation string `json:"operation"`
	Error     string `json:"error"`
	Timestamp int64  `json:"timestamp"`
}

// CalendarMonitor monitors calendar operations
type CalendarMonitor struct {
	errors []CalendarError
	logger *CalendarLogger
}

// NewCalendarMonitor creates a new calendar monitor
func NewCalendarMonitor() *CalendarMonitor {
	return &CalendarMonitor{
		errors: make([]CalendarError, 0),
		logger: &CalendarLogger{},
	}
}

// RecordError records a calendar operation error
func (cm *CalendarMonitor) RecordError(operation, errorStr string) {
	cm.errors = append(cm.errors, CalendarError{
		Operation: operation,
		Error:     errorStr,
		Timestamp: time.Now().Unix(),
	})
}

// GetErrors returns all recorded errors
func (cm *CalendarMonitor) GetErrors() []CalendarError {
	return cm.errors
}

// ClearErrors clears all recorded errors
func (cm *CalendarMonitor) ClearErrors() {
	cm.errors = make([]CalendarError, 0)
}

// CalendarMetrics tracks calendar operation metrics
type CalendarMetrics struct {
	TotalOperations int                    `json:"total_operations"`
	SuccessRate     float64               `json:"success_rate"`
	AverageDuration time.Duration         `json:"average_duration"`
	BookedAppointments int                 `json:"booked_appointments"`
	CancelledAppointments int              `json:"cancelled_appointments"`
}

// GetMetrics returns calendar operation metrics
func (cm *CalendarMonitor) GetMetrics() CalendarMetrics {
	// In a real implementation, this would aggregate from a database
	return CalendarMetrics{
		TotalOperations: 0,
		SuccessRate:     0.0,
		AverageDuration: 0,
		BookedAppointments: 0,
		CancelledAppointments: 0,
	}
}

// CalendarExecutor provides high-level calendar operations
type CalendarExecutor struct {
	manager *CalendarManager
	monitor *CalendarMonitor
	logger  *CalendarLogger
}

// NewCalendarExecutor creates a new calendar executor
func NewCalendarExecutor(calendarID string, credentials *CalendarCredentials) *CalendarExecutor {
	return &CalendarExecutor{
		manager: NewCalendarManager(calendarID, credentials),
		monitor: NewCalendarMonitor(),
		logger:  &CalendarLogger{},
	}
}

// BookAppointment books a new appointment
func (ce *CalendarExecutor) BookAppointment(ctx context.Context, sessionID string, summary, description string, start, end time.Time, attendees []string) (*CalendarEvent, error) {
	// Check availability first
	available, err := ce.manager.CheckAvailability(ctx, start, end)
	if err != nil {
		ce.monitor.RecordError("check_availability", err.Error())
		return nil, fmt.Errorf("failed to check availability: %w", err)
	}
	
	if !available {
		ce.logger.LogCalendarOperation("book_appointment", sessionID, false, map[string]interface{}{
			"reason": "slot_not_available",
			"start":  start.Format(time.RFC3339),
			"end":    end.Format(time.RFC3339),
		})
		return nil, fmt.Errorf("time slot is not available")
	}
	
	// Create attendees
	var eventAttendees []EventAttendee
	for _, email := range attendees {
		eventAttendees = append(eventAttendees, EventAttendee{Email: email})
	}
	
	// Create the event
	event := &CalendarEvent{
		Summary:     summary,
		Description: description,
		Start: EventDateTime{
			DateTime: start.Format(time.RFC3339),
			TimeZone: "Asia/Manila", // Assuming Philippine timezone
		},
		End: EventDateTime{
			DateTime: end.Format(time.RFC3339),
			TimeZone: "Asia/Manila",
		},
		Attendees: eventAttendees,
		Location:  "Virtual Meeting",
	}
	
	createdEvent, err := ce.manager.CreateEvent(ctx, event)
	if err != nil {
		ce.monitor.RecordError("create_event", err.Error())
		ce.logger.LogCalendarOperation("book_appointment", sessionID, false, map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to create appointment: %w", err)
	}
	
	ce.logger.LogCalendarOperation("book_appointment", sessionID, true, map[string]interface{}{
		"event_id": createdEvent.ID,
		"summary":  summary,
		"start":    start.Format(time.RFC3339),
		"end":      end.Format(time.RFC3339),
	})
	
	return createdEvent, nil
}

// GetAvailableSlots gets available time slots for a given date
func (ce *CalendarExecutor) GetAvailableSlots(ctx context.Context, sessionID string, date time.Time, duration time.Duration) ([]CalendarAvailability, error) {
	slots, err := ce.manager.FindAvailableSlots(ctx, date, duration)
	if err != nil {
		ce.monitor.RecordError("find_available_slots", err.Error())
		ce.logger.LogCalendarOperation("get_available_slots", sessionID, false, map[string]interface{}{
			"error": err.Error(),
			"date":  date.Format("2006-01-02"),
		})
		return nil, fmt.Errorf("failed to get available slots: %w", err)
	}
	
	ce.logger.LogCalendarOperation("get_available_slots", sessionID, true, map[string]interface{}{
		"date":       date.Format("2006-01-02"),
		"slot_count": len(slots),
	})
	
	return slots, nil
}

// HealthCheck checks if the calendar system is healthy
func (cm *CalendarManager) HealthCheck(ctx context.Context) error {
	// Try to list events to check connectivity
	now := time.Now()
	_, err := cm.ListEvents(ctx, now, now.Add(1*time.Hour))
	if err != nil {
		return fmt.Errorf("calendar system health check failed: %w", err)
	}
	return nil
}