package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateFolder(t *testing.T) {
	home, _ := os.UserHomeDir()
	downloads := t.TempDir()
	for folder, ok := range map[string]bool{
		downloads:                     true,
		"/":                           false,
		home:                          false,
		filepath.Dir(home):            false,
		"/System":                     false,
		filepath.Join(downloads, "x"): false,
	} {
		if err := ValidateFolder(folder); (err == nil) != ok {
			t.Errorf("ValidateFolder(%s) = %v", folder, err)
		}
	}
}
