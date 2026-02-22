package contract

import "encoding/json"

type CommandRequest[T any] struct {
	RequestID string `json:"requestId"`
	Data      T      `json:"data"`
}

type CommandError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CommandResponse[T any] struct {
	OK    bool          `json:"ok"`
	Data  *T            `json:"data,omitempty"`
	Error *CommandError `json:"error,omitempty"`
}

type Event[T any] struct {
	EventID    string `json:"eventId"`
	Type       string `json:"type"`
	OccurredAt string `json:"occurredAt"`
	Data       T      `json:"data"`
}

func ToJSON(value any) ([]byte, error) {
	return json.Marshal(value)
}

func FromJSON[T any](payload []byte) (T, error) {
	var result T
	err := json.Unmarshal(payload, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}
