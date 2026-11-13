package core

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const pollInterval = 2 * time.Second

type signature struct {
	size    int64
	modTime time.Time
}

// Watch は stop が閉じられるまで folder を見張り、Office ファイルからプロパティを消す。
// 書き込み中のファイルを触らないよう、大きさと更新時刻が1周回変わらなかったものだけを処理する。
func Watch(folder string, recursive bool, log func(string), stop <-chan struct{}) {
	if err := ValidateFolder(folder); err != nil {
		log("見張れません: " + err.Error())
		return
	}
	if recursive {
		log("見張り中（サブフォルダも含む）: " + folder)
	} else {
		log("見張り中: " + folder)
	}
	lastSeen := map[string]signature{}
	done := map[string]signature{}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		for _, path := range listTargets(folder, recursive, log) {
			sig, err := signatureOf(path)
			if err != nil || done[path] == sig {
				continue
			}
			if lastSeen[path] == sig {
				ScrubAndLog(path, log)
				// 失敗したファイルも、次に更新されるまでは触らない（開いている最中のファイルを毎回試さないため）
				if after, err := signatureOf(path); err == nil {
					done[path] = after
				}
			}
			lastSeen[path] = sig
		}
		select {
		case <-stop:
			log("見張りを止めました")
			return
		case <-ticker.C:
		}
	}
}

func ScrubAndLog(path string, log func(string)) {
	changed, err := ScrubFile(path)
	switch {
	case err != nil:
		log(fmt.Sprintf("スキップ: %s (%v)", filepath.Base(path), err))
	case changed:
		log("消しました: " + filepath.Base(path))
	default:
		log("消すものはありませんでした: " + filepath.Base(path))
	}
}

// ponytail: 毎周回フォルダ全体をたどる。ファイル数が数万を超えるフォルダで重くなるなら、OS のファイル変更通知に切り替える
func listTargets(folder string, recursive bool, log func(string)) []string {
	var targets []string
	if !recursive {
		entries, err := os.ReadDir(folder)
		if err != nil {
			log("フォルダを読めません: " + err.Error())
		}
		for _, e := range entries {
			if path := filepath.Join(folder, e.Name()); !e.IsDir() && IsTarget(path) {
				targets = append(targets, path)
			}
		}
		return targets
	}
	filepath.WalkDir(folder, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != folder && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if IsTarget(path) {
			targets = append(targets, path)
		}
		return nil
	})
	return targets
}

func signatureOf(path string) (signature, error) {
	info, err := os.Stat(path)
	if err != nil {
		return signature{}, err
	}
	return signature{info.Size(), info.ModTime()}, nil
}
