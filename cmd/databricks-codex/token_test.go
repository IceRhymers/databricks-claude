package main

import (
	"testing"
)

// TestConstructGatewayURL: verifies the host-relative AI Gateway URL format.
func TestConstructGatewayURL(t *testing.T) {
	tests := []struct {
		name string
		host string
		want string
	}{
		{
			name: "plain host",
			host: "https://dbc-abc123.cloud.databricks.com",
			want: "https://dbc-abc123.cloud.databricks.com/ai-gateway/openai/v1",
		},
		{
			name: "host with trailing slash is trimmed",
			host: "https://dbc-abc123.cloud.databricks.com/",
			want: "https://dbc-abc123.cloud.databricks.com/ai-gateway/openai/v1",
		},
		{
			name: "host with multiple trailing slashes is trimmed",
			host: "https://dbc-abc123.cloud.databricks.com///",
			want: "https://dbc-abc123.cloud.databricks.com/ai-gateway/openai/v1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ConstructGatewayURL(tc.host)
			if got != tc.want {
				t.Errorf("ConstructGatewayURL(%q) = %q, want %q", tc.host, got, tc.want)
			}
		})
	}
}
