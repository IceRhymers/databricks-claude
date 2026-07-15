package proxy

import "os"

// getuid is overridable in tests.
var getuid = os.Getuid

// SecurityChecks returns a list of human-readable warnings for risky startup
// conditions. Returns nil when no issues are detected.
func SecurityChecks() []string {
	var warnings []string
	if getuid() == 0 {
		warnings = append(warnings, "WARNING: running as root is not recommended for proxy operation")
	}
	if w := umaskWarning(); w != "" {
		warnings = append(warnings, w)
	}
	return warnings
}
