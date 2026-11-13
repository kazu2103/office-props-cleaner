//go:build windows

// Windows のウィンドウとタスクトレイ。build.sh でマニフェストを埋め込んでからビルドする。
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"

	"github.com/kazu2103/office-props-cleaner/core"
)

const title = "Officeプロパティ消去"

type app struct {
	mu       sync.Mutex
	settings core.Settings
	stop     chan struct{}
	quitting bool

	window     *walk.MainWindow
	tray       *walk.NotifyIcon
	folder     *walk.Label
	status     *walk.Label
	toggle     *walk.PushButton
	autostart  *walk.CheckBox
	recursive  *walk.CheckBox
	logView    *walk.TextEdit
	trayToggle *walk.Action
}

func main() {
	watchOnLaunch := slices.Contains(os.Args[1:], core.WatchOnLaunchArg)
	if !acquireSingleInstance() {
		if !watchOnLaunch {
			focusRunningInstance()
		}
		return
	}
	a := &app{settings: core.LoadSettings()}

	if err := (MainWindow{
		AssignTo: &a.window,
		Title:    title,
		MinSize:  Size{Width: 520, Height: 360},
		Size:     Size{Width: 600, Height: 420},
		Visible:  !watchOnLaunch,
		Layout:   VBox{},
		Children: []Widget{
			Composite{Layout: HBox{MarginsZero: true}, Children: []Widget{
				Label{Text: "見張るフォルダ:"},
				Label{AssignTo: &a.folder, Text: a.settings.Folder},
				HSpacer{},
				PushButton{Text: "変更…", OnClicked: a.chooseFolder},
			}},
			Composite{Layout: HBox{MarginsZero: true}, Children: []Widget{
				PushButton{AssignTo: &a.toggle, OnClicked: a.toggleWatching},
				Label{AssignTo: &a.status},
				HSpacer{},
				PushButton{Text: "ファイルを選んで今すぐ消す…", OnClicked: a.scrubChosen},
			}},
			CheckBox{AssignTo: &a.recursive, Text: "サブフォルダも見張る", Checked: a.settings.Recursive, OnClicked: a.toggleRecursive},
			CheckBox{AssignTo: &a.autostart, Text: "ログイン時に自動で見張りを始める", Checked: a.settings.Autostart, OnClicked: a.toggleAutostart},
			TextEdit{AssignTo: &a.logView, ReadOnly: true, VScroll: true},
			Composite{Layout: HBox{MarginsZero: true}, Children: []Widget{
				Label{Text: "ウィンドウを閉じても、タスクトレイで動き続けます"},
				HSpacer{},
				PushButton{Text: "終了", OnClicked: a.quit},
			}},
		},
	}).Create(); err != nil {
		walk.MsgBox(nil, title, err.Error(), walk.MsgBoxIconError)
		return
	}

	a.window.Closing().Attach(func(canceled *bool, _ walk.CloseReason) {
		if !a.quitting {
			*canceled = true
			a.window.Hide()
		}
	})
	if err := a.setupTray(); err != nil {
		a.log("タスクトレイのアイコンを出せません: " + err.Error())
	}

	if watchOnLaunch {
		a.start()
	} else {
		a.showStopped()
	}
	a.window.Run()
}

func (a *app) setupTray() error {
	icon, err := loadIcon()
	if err != nil {
		return err
	}
	a.window.SetIcon(icon)
	a.tray, err = walk.NewNotifyIcon(a.window)
	if err != nil {
		return err
	}
	a.tray.SetIcon(icon)
	a.tray.SetToolTip(title)
	a.tray.MouseDown().Attach(func(_, _ int, button walk.MouseButton) {
		if button == walk.LeftButton {
			a.showWindow()
		}
	})
	menu := a.tray.ContextMenu().Actions()
	open := walk.NewAction()
	open.SetText("ウィンドウを開く")
	open.Triggered().Attach(a.showWindow)
	a.trayToggle = walk.NewAction()
	a.trayToggle.Triggered().Attach(a.toggleWatching)
	quit := walk.NewAction()
	quit.SetText("終了")
	quit.Triggered().Attach(a.quit)
	for _, action := range []*walk.Action{open, a.trayToggle, walk.NewSeparatorAction(), quit} {
		menu.Add(action)
	}
	return a.tray.SetVisible(true)
}

// walk はアイコンをファイルかリソースからしか読めないため、埋め込んだ ico を一時ファイルに書き出して読む
func loadIcon() (*walk.Icon, error) {
	path := filepath.Join(os.TempDir(), "office-props-cleaner.ico")
	if err := os.WriteFile(path, iconBytes, 0o644); err != nil {
		return nil, err
	}
	return walk.NewIconFromFile(path)
}

func (a *app) showWindow() {
	a.window.Show()
	win.ShowWindow(a.window.Handle(), win.SW_RESTORE)
	win.SetForegroundWindow(a.window.Handle())
}

func (a *app) watching() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.stop != nil
}

func (a *app) start() {
	a.mu.Lock()
	if a.stop != nil {
		close(a.stop)
	}
	a.stop = make(chan struct{})
	go core.Watch(a.settings.Folder, a.settings.Recursive, a.log, a.stop)
	a.mu.Unlock()
	a.setStatus("● 見張り中", "見張りを止める")
}

func (a *app) halt() {
	a.mu.Lock()
	if a.stop != nil {
		close(a.stop)
		a.stop = nil
	}
	a.mu.Unlock()
	a.showStopped()
}

func (a *app) showStopped() {
	a.setStatus("○ 停止中", "見張りを始める")
}

func (a *app) setStatus(status, toggle string) {
	a.status.SetText(status)
	a.toggle.SetText(toggle)
	if a.trayToggle != nil {
		a.trayToggle.SetText(toggle)
	}
}

func (a *app) toggleWatching() {
	if a.watching() {
		a.halt()
	} else {
		a.start()
	}
}

// 見張りは別の goroutine から呼ぶので、画面の更新はウィンドウのスレッドに渡す
func (a *app) log(message string) {
	a.window.Synchronize(func() {
		a.logView.AppendText(strings.TrimSpace(message) + "\r\n")
	})
}

func (a *app) save() {
	if err := core.SaveSettings(a.settings); err != nil {
		a.log("設定を保存できません: " + err.Error())
	}
}

func (a *app) chooseFolder() {
	dlg := walk.FileDialog{Title: "見張るフォルダを選ぶ", InitialDirPath: a.settings.Folder}
	if ok, err := dlg.ShowBrowseFolder(a.window); err != nil || !ok {
		return
	}
	if err := core.ValidateFolder(dlg.FilePath); err != nil {
		walk.MsgBox(a.window, "このフォルダは選べません", err.Error(), walk.MsgBoxIconWarning)
		return
	}
	a.settings.Folder = dlg.FilePath
	a.save()
	a.folder.SetText(dlg.FilePath)
	if a.watching() {
		a.start()
	}
}

func (a *app) scrubChosen() {
	dlg := walk.FileDialog{
		Title:  "プロパティを消すファイルを選ぶ",
		Filter: "Office ファイル|*.xlsx;*.xlsm;*.docx;*.docm;*.pptx;*.pptm",
	}
	if ok, err := dlg.ShowOpenMultiple(a.window); err != nil || !ok {
		return
	}
	for _, path := range dlg.FilePaths {
		core.ScrubAndLog(path, a.log)
	}
}

func (a *app) toggleRecursive() {
	a.settings.Recursive = a.recursive.Checked()
	a.save()
	if a.watching() {
		a.start()
	}
}

func (a *app) toggleAutostart() {
	enabled := a.autostart.Checked()
	if err := core.SetAutostart(enabled); err != nil {
		a.log(fmt.Sprintf("自動起動を設定できません: %v", err))
		a.autostart.SetChecked(!enabled)
		return
	}
	a.settings.Autostart = enabled
	a.save()
}

func (a *app) quit() {
	a.halt()
	a.quitting = true
	if a.tray != nil {
		a.tray.Dispose()
	}
	a.window.Close()
}
