package core

import (
	"os"

	"golang.org/x/sys/windows/registry"
)

const runValueName = "OfficePropsCleaner"

func SetAutostart(enabled bool) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	if !enabled {
		if err := key.DeleteValue(runValueName); err != nil && err != registry.ErrNotExist {
			return err
		}
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return key.SetStringValue(runValueName, `"`+exe+`" `+WatchOnLaunchArg)
}
