package core

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
)

const launchAgentLabel = "jp.ascend.office-props-cleaner"

func SetAutostart(enabled bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	plist := filepath.Join(home, "Library", "LaunchAgents", launchAgentLabel+".plist")
	if !enabled {
		if err := os.Remove(plist); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	body := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>%s</string>
<key>ProgramArguments</key><array><string>%s</string></array>
<key>RunAtLoad</key><true/>
</dict></plist>
`, launchAgentLabel, html.EscapeString(exe))
	if err := os.MkdirAll(filepath.Dir(plist), 0o755); err != nil {
		return err
	}
	return os.WriteFile(plist, []byte(body), 0o644)
}
