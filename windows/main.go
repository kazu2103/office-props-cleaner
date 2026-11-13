//go:build windows

// Windows のタスクトレイに常駐する画面。go build -ldflags -H=windowsgui でビルドする。
package main

import (
	"fmt"
	"sync"

	"github.com/getlantern/systray"
	"github.com/sqweek/dialog"

	"github.com/kazu2103/office-props-cleaner/core"
)

const recentLogCount = 10

type tray struct {
	mu        sync.Mutex
	settings  core.Settings
	stop      chan struct{}
	status    *systray.MenuItem
	toggle    *systray.MenuItem
	folder    *systray.MenuItem
	autostart *systray.MenuItem
	logs      []*systray.MenuItem
	recent    []string
}

func main() {
	systray.Run(onReady, func() {})
}

func onReady() {
	t := &tray{settings: core.LoadSettings()}
	systray.SetIcon(icon)
	systray.SetTooltip("Officeプロパティ消去")

	t.status = systray.AddMenuItem("", "")
	t.status.Disable()
	t.toggle = systray.AddMenuItem("", "")
	systray.AddSeparator()
	t.folder = systray.AddMenuItem("", "")
	t.folder.Disable()
	chooseFolder := systray.AddMenuItem("フォルダを変更…", "")
	scrubChosen := systray.AddMenuItem("ファイルを選んで今すぐ消す…", "")
	t.autostart = systray.AddMenuItemCheckbox("ログイン時に自動で始める", "", t.settings.Autostart)
	systray.AddSeparator()
	header := systray.AddMenuItem("最近の処理", "")
	header.Disable()
	for i := 0; i < recentLogCount; i++ {
		item := systray.AddMenuItem("", "")
		item.Disable()
		item.Hide()
		t.logs = append(t.logs, item)
	}
	systray.AddSeparator()
	quit := systray.AddMenuItem("終了", "")

	t.refreshFolder()
	t.start()

	go func() {
		for {
			select {
			case <-t.toggle.ClickedCh:
				if t.watching() {
					t.halt()
				} else {
					t.start()
				}
			case <-chooseFolder.ClickedCh:
				t.chooseFolder()
			case <-scrubChosen.ClickedCh:
				t.scrubChosen()
			case <-t.autostart.ClickedCh:
				t.toggleAutostart()
			case <-quit.ClickedCh:
				t.halt()
				systray.Quit()
				return
			}
		}
	}()
}

func (t *tray) watching() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.stop != nil
}

func (t *tray) start() {
	t.mu.Lock()
	if t.stop != nil {
		close(t.stop)
	}
	t.stop = make(chan struct{})
	go core.Watch(t.settings.Folder, t.log, t.stop)
	t.mu.Unlock()
	t.status.SetTitle("● 見張り中")
	t.toggle.SetTitle("見張りを止める")
}

func (t *tray) halt() {
	t.mu.Lock()
	if t.stop != nil {
		close(t.stop)
		t.stop = nil
	}
	t.mu.Unlock()
	t.status.SetTitle("○ 停止中")
	t.toggle.SetTitle("見張りを始める")
}

func (t *tray) log(message string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.recent = append([]string{message}, t.recent...)
	if len(t.recent) > recentLogCount {
		t.recent = t.recent[:recentLogCount]
	}
	for i, item := range t.logs {
		if i < len(t.recent) {
			item.SetTitle(t.recent[i])
			item.Show()
		}
	}
}

func (t *tray) refreshFolder() {
	t.folder.SetTitle("見張るフォルダ: " + t.settings.Folder)
}

func (t *tray) save() {
	if err := core.SaveSettings(t.settings); err != nil {
		t.log("設定を保存できません: " + err.Error())
	}
}

func (t *tray) chooseFolder() {
	folder, err := dialog.Directory().SetStartDir(t.settings.Folder).Title("見張るフォルダを選ぶ").Browse()
	if err != nil {
		return
	}
	t.settings.Folder = folder
	t.save()
	t.refreshFolder()
	if t.watching() {
		t.start()
	}
}

func (t *tray) scrubChosen() {
	path, err := dialog.File().Filter("Office ファイル", "xlsx", "xlsm", "docx", "docm", "pptx", "pptm").Title("プロパティを消すファイルを選ぶ").Load()
	if err != nil {
		return
	}
	core.ScrubAndLog(path, t.log)
}

func (t *tray) toggleAutostart() {
	enabled := !t.settings.Autostart
	if err := core.SetAutostart(enabled); err != nil {
		t.log(fmt.Sprintf("自動起動を設定できません: %v", err))
		return
	}
	t.settings.Autostart = enabled
	t.save()
	if enabled {
		t.autostart.Check()
	} else {
		t.autostart.Uncheck()
	}
}
