package core

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const pollInterval = 2 * time.Second

type signature struct {
	size    int64
	modTime time.Time
}

// Watch は stop が閉じられるまで folder を見張り、Office ファイルからプロパティを消す。
// 書き込み中のファイルを触らないよう、大きさと更新時刻が1周回変わらなかったものだけを処理する。
func Watch(folder string, log func(string), stop <-chan struct{}) {
	log("見張り中: " + folder)
	lastSeen := map[string]signature{}
	done := map[string]signature{}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		entries, err := os.ReadDir(folder)
		if err != nil {
			log("フォルダを読めません: " + err.Error())
		}
		for _, e := range entries {
			path := filepath.Join(folder, e.Name())
			if e.IsDir() || !IsTarget(path) {
				continue
			}
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
	}
}

func signatureOf(path string) (signature, error) {
	info, err := os.Stat(path)
	if err != nil {
		return signature{}, err
	}
	return signature{info.Size(), info.ModTime()}, nil
}
