// Package browser opens URLs in the user's default browser, best-effort.
package browser

import (
	"os/exec"
	"runtime"
)

// Open attempts to open url. Errors are returned but commonly ignored by
// callers (they print the URL as a fallback).
func Open(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default: // linux, *bsd
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
