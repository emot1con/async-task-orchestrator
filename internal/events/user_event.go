package events

import "time"

// UserRegisteredEvent - emitted when user registers
type UserRegisteredEvent struct {
	BaseEvent
	Data     UserRegisteredData `json:"data"`
	Metadata Metadata           `json:"metadata"`
}

type UserRegisteredData struct {
	UserID       int       `json:"user_id"`
	Username     string    `json:"username"`
	Email        string    `json:"email,omitempty"`
	RegisteredAt time.Time `json:"registered_at"`
}

// UserLoggedInEvent - emitted when user logs in
type UserLoggedInEvent struct {
	BaseEvent
	Data     UserLoggedInData `json:"data"`
	Metadata Metadata         `json:"metadata"`
}

type UserLoggedInData struct {
	UserID     int       `json:"user_id"`
	Username   string    `json:"username"`
	IPAddress  string    `json:"ip_address,omitempty"`
	UserAgent  string    `json:"user_agent,omitempty"`
	LoggedInAt time.Time `json:"logged_in_at"`
}
