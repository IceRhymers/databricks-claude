package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// readDatabricksCfgHost reads the host for the given profile from ~/.databrickscfg.
// Returns empty string if not found or file unreadable.
func readDatabricksCfgHost(profile string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	f, err := os.Open(filepath.Join(home, ".databrickscfg"))
	if err != nil {
		return ""
	}
	defer f.Close()

	target := "[" + profile + "]"
	inSection := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") {
			inSection = strings.EqualFold(line, target)
			continue
		}
		if inSection {
			k, v, ok := strings.Cut(line, "=")
			if ok && strings.TrimSpace(k) == "host" {
				return strings.TrimSpace(v)
			}
		}
	}
	return ""
}
