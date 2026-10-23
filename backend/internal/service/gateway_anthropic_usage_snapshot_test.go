package service

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestParseSSEUsagePassthroughFailureSnapshot(t *testing.T) {
	usage := &ClaudeUsage{}
	parseSSEUsagePassthrough(`{"type":"message_start","message":{"usage":{"input_tokens":45998,"output_tokens":1,"cache_creation_input_tokens":40,"cache_creation":{"ephemeral_5m_input_tokens":40}}}}`, usage)
	parseSSEUsagePassthrough(`{"type":"message_delta","delta":{"stop_reason":null},"usage":{"usage_snapshot":true,"input_tokens":0,"output_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}`, usage)
	require.Equal(t, ClaudeUsage{}, *usage, "failed preparation must clear the initial estimates, including cache detail")
	parseSSEUsagePassthrough(`{"type":"message_delta","delta":{"stop_reason":null},"usage":{"usage_snapshot":true,"input_tokens":45998,"output_tokens":1500,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}`, usage)
	require.Equal(t, 45998, usage.InputTokens)
	require.Equal(t, 1500, usage.OutputTokens, "interrupted long output must retain its accumulated usage")
	parseSSEUsagePassthrough(`{"type":"error","error":{"type":"api_error","message":"upstream overloaded"}}`, usage)
	require.Equal(t, 1500, usage.OutputTokens)
}

func TestParseSSEUsagePassthroughSnapshotRequiresCompleteValidCounters(t *testing.T) {
	for _, payload := range []string{
		`{"type":"message_delta","usage":{"usage_snapshot":true,"input_tokens":200}}`,
		`{"type":"message_delta","usage":{"usage_snapshot":true,"input_tokens":0,"output_tokens":-1,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}`,
		`{"type":"message_delta","usage":{"usage_snapshot":true,"input_tokens":0,"output_tokens":0.5,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}`,
		`{"type":"message_delta","usage":{"usage_snapshot":true,"input_tokens":0,"output_tokens":"0","cache_creation_input_tokens":0,"cache_read_input_tokens":0}}`,
		`{"type":"message_delta","usage":{"input_tokens":0,"output_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}`,
	} {
		usage := &ClaudeUsage{InputTokens: 100, OutputTokens: 5}
		parseSSEUsagePassthrough(payload, usage)
		require.Equal(t, 100, usage.InputTokens, payload)
		require.Equal(t, 5, usage.OutputTokens, payload)
	}
}
