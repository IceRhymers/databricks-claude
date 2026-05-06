package proxy

import "regexp"

var sensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)Bearer [^\s"]+`),
	regexp.MustCompile(`dapi[a-zA-Z0-9-]+`),
	regexp.MustCompile(`(?i)x-api-key:\s*[^\s"]+`),
	regexp.MustCompile(`(?i)x-databricks-authorization:\s*\S+`),
	regexp.MustCompile(`(?i)Authorization:\s*Basic\s+[A-Za-z0-9+/=]+`),
	regexp.MustCompile(`"access_token"\s*:\s*"[^"]+"`),
	regexp.MustCompile(`DATABRICKS_TOKEN=[^\s&"]+`),
}

// sanitizeLogOutput redacts sensitive values (Bearer tokens, PATs, API keys)
// from strings before they are written to logs.
func sanitizeLogOutput(s string) string {
	for _, re := range sensitivePatterns {
		s = re.ReplaceAllString(s, "[REDACTED]")
	}
	return s
}
