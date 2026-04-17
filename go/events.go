// Package gobutton provides a Go client for button-hub, a REST API for reacting to Zigbee button presses.
//
// The package offers two levels of abstraction:
//   - [Listen]: high-level function with automatic registration, reconnection, and cleanup
//   - [Client.Listen]: low-level SSE stream client for custom retry logic
package gobutton

// ActionType represents a button press action.
type ActionType string

const (
	ActionSingle    ActionType = "Single"
	ActionDouble    ActionType = "Double"
	ActionLongPress ActionType = "LongPress"
)

// Event mirrors the JSON schema emitted by button-hub.
type Event struct {
	// ButtonID is the unique identifier of the button that generated the event.
	ButtonID string `json:"button_id"`
	// Action is the type of button press: Single, Double, or LongPress.
	// Note: "hold" and "long" from Zigbee2MQTT are normalized to LongPress.
	Action ActionType `json:"action"`
	// Battery is the current battery level as a percentage (0-100), or nil if unknown.
	Battery *uint8 `json:"battery,omitempty"`
	// Timestamp is an ISO 8601 formatted time string when the event occurred.
	Timestamp string `json:"timestamp"`
}
