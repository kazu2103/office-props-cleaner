package core

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// 中のファイルを一斉に書き換えると被害が大きいフォルダ。ドライブやディスクの直下・ホームフォルダそのもの・OS とアプリの置き場所
var tooBroadFolders = map[string][]string{
	"darwin":  {"/Users", "/System", "/Library", "/Applications", "/private", "/Volumes", "/opt", "/usr"},
	"windows": {`\Users`, `\Windows`, `\Program Files`, `\Program Files (x86)`, `\ProgramData`},
}

func ValidateFolder(folder string) error {
	if folder == "" {
		return errors.New("フォルダが指定されていません")
	}
	clean, err := filepath.Abs(folder)
	if err != nil {
		return err
	}
	if info, err := os.Stat(clean); err != nil || !info.IsDir() {
		return errors.New("フォルダが見つかりません: " + clean)
	}
	if filepath.Dir(clean) == clean {
		return errors.New("ドライブやディスクの直下は指定できません: " + clean)
	}
	if home, err := os.UserHomeDir(); err == nil && samePath(clean, home) {
		return errors.New("ホームフォルダそのものは指定できません。ダウンロードなど、その中のフォルダを選んでください")
	}
	volume := filepath.VolumeName(clean)
	for _, broad := range tooBroadFolders[runtime.GOOS] {
		if samePath(clean, volume+broad) {
			return errors.New("システムやアプリのフォルダは指定できません: " + clean)
		}
	}
	return nil
}

func samePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
