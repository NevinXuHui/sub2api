package service

import "testing"

func TestAnthropicSSEHasContent(t *testing.T) {
	for _, tc := range []struct {
		data string
		want bool
	}{
		{`{"type":"message_start","message":{"content":[]}}`, false},
		{`{"type":"ping"}`, false},
		{`{"type":"content_block_start","content_block":{"type":"tool_use","input":{}}}`, false},
		{`{"type":"content_block_delta","delta":{"type":"text_delta","text":""}}`, false},
		{`{"type":"content_block_delta","delta":{"type":"signature_delta","signature":"abc"}}`, false},
		{`{"type":"content_block_delta","delta":{"type":"text_delta","text":"hi"}}`, true},
		{`{"type":"content_block_delta","delta":{"type":"thinking_delta","thinking":"think"}}`, true},
		{`{"type":"content_block_delta","delta":{"type":"input_json_delta","partial_json":"{"}}`, true},
		{`{"type":"content_block_start","content_block":{"type":"text","text":"hi"}}`, true},
		{`{"type":"message_delta","usage":{"output_tokens":5}}`, false},
		{`[DONE]`, false}, {`broken`, false},
	} {
		if got := anthropicSSEHasContent(tc.data); got != tc.want {
			t.Errorf("%s: got %v want %v", tc.data, got, tc.want)
		}
	}
}
