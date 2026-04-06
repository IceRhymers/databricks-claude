package proxy

import (
	"fmt"
	"os"
	"syscall"
)

// getuid is overridable in tests.
var getuid = os.Getuid

// SecurityChecks returns a list of human-readable warnings for risky startup
// conditions. Returns nil when no issues are detected.
func SecurityChecks() []string {
	var warnings []string

	if getuid() == 0 {
		warnings = append(warnings, "WARNING: running as root is not recommended for proxy operation")
	}

	// Check if umask allows group or other write
	old := syscall.Umask(0)
	syscall.Umask(old)
	if old&0o022 != 0o022 {
		warnings = append(warnings, fmt.Sprintf("WARNING: umask %04o allows group/other write — config files may be world-writable", old))
	}

	return warnings
}
