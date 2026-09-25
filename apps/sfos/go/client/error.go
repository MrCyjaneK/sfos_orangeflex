package client

import (
	"encoding/json"
	"fmt"
)

// APIError is a non-success HTTP status from a Flex gateway.
type APIError struct {
	URL     string
	Status  int
	Code    string
	Message string
	Body    []byte
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("client: %s: HTTP %d %s", e.URL, e.Status, e.Message)
	}
	if e.Code != "" {
		return fmt.Sprintf("client: %s: HTTP %d %s", e.URL, e.Status, e.Code)
	}
	return fmt.Sprintf("client: %s: HTTP %d", e.URL, e.Status)
}

func apiErr(rawURL string, status int, body []byte) error {
	e := &APIError{URL: rawURL, Status: status, Body: body}
	var w struct {
		Code             string `json:"code"`
		Message          string `json:"message"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if json.Unmarshal(body, &w) == nil {
		e.Code = w.Code
		if w.Code == "" {
			e.Code = w.Error
		}
		e.Message = w.Message
		if e.Message == "" {
			e.Message = w.ErrorDescription
		}
	}
	return e
}
