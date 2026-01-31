package events

import (
	"github.com/google/uuid"
	"time"
)

// BaseEvent is embedded in all events
type BaseEvent struct {
	EventID      string    `json:"event_id"`
	EventType    string    `json:"event_type"`
	EventVersion string    `json:"event_version"`
	Timestamp    time.Time `json:"timestamp"`
	Source       string    `json:"source"`
}

// Metadata contains contextual info
type Metadata struct {
	CorrelationID string `json:"correlation_id,omitempty"`
	RequestID     string `json:"request_id,omitempty"`
	WorkerID      string `json:"worker_id,omitempty"`
	UserAgent     string `json:"user_agent,omitempty"`
	IPAddress     string `json:"ip_address,omitempty"`
}

// NewBaseEvent creates base event with defaults
func NewBaseEvent(eventType, source string) BaseEvent {
	return BaseEvent{
		EventID:      uuid.New().String(),
		EventType:    eventType,
		EventVersion: EventVersionV1,
		Timestamp:    time.Now().UTC(),
		Source:       source,
	}
}
