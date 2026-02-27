package contract

import "encoding/json"

const (
	SubjectUserCommandCreate = "user.command.create"
	SubjectUserCommandList   = "user.command.list"
	SubjectUserCommandGet    = "user.command.get"
	SubjectUserCommandUpdate = "user.command.update"
	SubjectUserCommandDelete = "user.command.delete"

	SubjectUserEventCreated = "user.event.created"
	SubjectUserEventUpdated = "user.event.updated"
	SubjectUserEventDeleted = "user.event.deleted"
)

type CommandRequest[T any] struct { // T is a generic type parameter that allows CommandRequest to be used with any data type
	RequestID string `json:"requestId"`
	Data      T      `json:"data"`
}

type CommandError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CommandResponse[T any] struct { // T is a generic type parameter that allows CommandResponse to be used with any data type
	OK    bool          `json:"ok"`
	Data  *T            `json:"data,omitempty"`
	Error *CommandError `json:"error,omitempty"`
}

type Event[T any] struct { // T is a generic type parameter that allows Event to be used with any data type
	EventID    string `json:"eventId"`
	Type       string `json:"type"`
	OccurredAt string `json:"occurredAt"`
	Data       T      `json:"data"`
}

// marshal JSON payloads for NATS messages
func ToJSON(value any) ([]byte, error) {
	return json.Marshal(value)
}

// unmarshal JSON payload into a struct of type T
func FromJSON[T any](payload []byte) (T, error) {
	var result T
	err := json.Unmarshal(payload, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}
