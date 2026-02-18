package ws

import "encoding/json"

type RequestMessage struct {
	RequestID string          `json:"requestId"`
	Action    string          `json:"action"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type ErrorMessage struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ResponseMessage struct {
	RequestID string        `json:"requestId,omitempty"`
	OK        bool          `json:"ok"`
	Data      any           `json:"data,omitempty"`
	Error     *ErrorMessage `json:"error,omitempty"`
}

type IDPayload struct {
	ID string `json:"id"`
}

type UpdatePayload struct {
	ID        string  `json:"id"`
	FirstName *string `json:"firstName"`
	LastName  *string `json:"lastName"`
	Email     *string `json:"email"`
	Phone     *string `json:"phone"`
	Age       *int32  `json:"age"`
	Status    *string `json:"status"`
}
