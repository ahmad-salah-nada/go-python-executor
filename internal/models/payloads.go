package models

// RequestPayload represents the incoming request for code execution
type RequestPayload struct {
	ID   string `json:"id,omitempty"`
	Code string `json:"code"`
}

// ResponsePayload represents the execution result
type ResponsePayload struct {
	ID     string `json:"id,omitempty"`
	Stdout string `json:"stdout,omitempty"`
	Stderr string `json:"stderr,omitempty"`
	Error  string `json:"error,omitempty"`
}
