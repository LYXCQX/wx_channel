package ws

import "encoding/json"

type MessageType string

const (
	MsgTypeHeartbeat MessageType = "heartbeat"
	MsgTypeCommand   MessageType = "command"
	MsgTypeResponse  MessageType = "response"
	MsgTypeBind      MessageType = "bind"
)

type CloudMessage struct {
	ID         string          `json:"id"`
	Type       MessageType     `json:"type"`
	ClientID   string          `json:"client_id"`
	Payload    json.RawMessage `json:"payload"`
	Timestamp  int64           `json:"timestamp"`
	Compressed bool            `json:"compressed,omitempty"`
}

type HeartbeatPayload struct {
	Hostname            string `json:"hostname"`
	Version             string `json:"version"`
	Status              string `json:"status"`
	HardwareFingerprint string `json:"hardware_fingerprint,omitempty"` // JSON string of hardware fingerprint
}

type CommandPayload struct {
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

type ResponsePayload struct {
	RequestID string          `json:"request_id"`
	Success   bool            `json:"success"`
	Data      json.RawMessage `json:"data"`
	Error     string          `json:"error"`
}
