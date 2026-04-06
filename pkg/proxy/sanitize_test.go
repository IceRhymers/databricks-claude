package proxy

import "testing"

func TestSanitizeLogOutput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "bearer token in body",
			input: `{"Authorization": "Bearer dapi1234567890abcdef"}`,
			want:  `{"Authorization": "[REDACTED]"}`,
		},
		{
			name:  "dapi PAT token",
			input: `token was dapi-abc123XYZ in the request`,
			want:  `token was [REDACTED] in the request`,
		},
		{
			name:  "x-api-key header case insensitive",
			input: `X-Api-Key: secret-key-value`,
			want:  `[REDACTED]`,
		},
		{
			name:  "x-api-key lowercase",
			input: `x-api-key: another-secret`,
			want:  `[REDACTED]`,
		},
		{
			name:  "safe string unchanged",
			input: `{"status": "ok", "code": 200}`,
			want:  `{"status": "ok", "code": 200}`,
		},
		{
			name:  "multiple sensitive values",
			input: `Bearer tok123 and dapi-xyz456`,
			want:  `[REDACTED] and [REDACTED]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeLogOutput(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeLogOutput(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
