//go:build !windows

package proxy

import (
	"fmt"
	"syscall"
)

func umaskWarning() string {
	old := syscall.Umask(0)
	syscall.Umask(old)
	if old&0o022 != 0o022 {
		return fmt.Sprintf("WARNING: umask %04o allows group/other write — config files may be world-writable", old)
	}
	return ""
}
