package service

import "encoding/json"

// anthropicSSEHasContent excludes message_start, ping, empty tool headers and usage.
// Count text, thinking or actual tool argument bytes, matching the Kiro bridge.
func anthropicSSEHasContent(data string) bool {
	var event struct {
		Type  string `json:"type"`
		Delta struct {
			Type        string `json:"type"`
			Text        string `json:"text"`
			Thinking    string `json:"thinking"`
			PartialJSON string `json:"partial_json"`
		} `json:"delta"`
		ContentBlock struct {
			Type     string `json:"type"`
			Text     string `json:"text"`
			Thinking string `json:"thinking"`
		} `json:"content_block"`
	}
	if json.Unmarshal([]byte(data), &event) != nil {
		return false
	}
	switch event.Type {
	case "content_block_delta":
		switch event.Delta.Type {
		case "text_delta":
			return event.Delta.Text != ""
		case "thinking_delta":
			return event.Delta.Thinking != ""
		case "input_json_delta":
			return event.Delta.PartialJSON != ""
		}
	case "content_block_start":
		return event.ContentBlock.Text != "" || event.ContentBlock.Thinking != ""
	}
	return false
}
