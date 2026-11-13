package core

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Settings struct {
	Folder    string `json:"folder"`
	Autostart bool   `json:"autostart"`
	Recursive bool   `json:"recursive"`
}

func settingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".office-props-cleaner.json")
}

func LoadSettings() Settings {
	home, _ := os.UserHomeDir()
	s := Settings{Folder: filepath.Join(home, "Downloads")}
	if data, err := os.ReadFile(settingsPath()); err == nil {
		_ = json.Unmarshal(data, &s)
	}
	return s
}

func SaveSettings(s Settings) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath(), data, 0o644)
}
