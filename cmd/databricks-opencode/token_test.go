package main

import (
	"testing"
)

func TestConstructGatewayURL(t *testing.T) {
	tests := []struct {
		name string
		host string
		want string
	}{
		{
			name: "plain host",
			host: "https://dbc-abc123.cloud.databricks.com",
			want: "https://dbc-abc123.cloud.databricks.com/ai-gateway/anthropic",
		},
		{
			name: "trailing slash trimmed",
			host: "https://dbc-abc123.cloud.databricks.com/",
			want: "https://dbc-abc123.cloud.databricks.com/ai-gateway/anthropic",
		},
		{
			name: "multiple trailing slashes trimmed",
			host: "https://example.databricks.com///",
			want: "https://example.databricks.com/ai-gateway/anthropic",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ConstructGatewayURL(tc.host)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestConstructGeminiGatewayURL(t *testing.T) {
	tests := []struct {
		name string
		host string
		want string
	}{
		{
			name: "plain host",
			host: "https://dbc-abc123.cloud.databricks.com",
			want: "https://dbc-abc123.cloud.databricks.com/ai-gateway/gemini/v1beta",
		},
		{
			name: "trailing slash trimmed",
			host: "https://dbc-abc123.cloud.databricks.com/",
			want: "https://dbc-abc123.cloud.databricks.com/ai-gateway/gemini/v1beta",
		},
		{
			name: "multiple trailing slashes trimmed",
			host: "https://example.databricks.com///",
			want: "https://example.databricks.com/ai-gateway/gemini/v1beta",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ConstructGeminiGatewayURL(tc.host)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
