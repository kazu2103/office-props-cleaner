//go:build windows

package main

import (
	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

// プロセスが終わるまで握り続けるので、解放しない
var instanceMutex windows.Handle

func acquireSingleInstance() bool {
	name, _ := windows.UTF16PtrFromString(`Local\io.github.kazu2103.office-props-cleaner`)
	handle, err := windows.CreateMutex(nil, false, name)
	if err == windows.ERROR_ALREADY_EXISTS {
		windows.CloseHandle(handle)
		return false
	}
	instanceMutex = handle
	return err == nil
}

func focusRunningInstance() {
	titlePtr, _ := windows.UTF16PtrFromString(title)
	hwnd := win.FindWindow(nil, titlePtr)
	if hwnd == 0 {
		return
	}
	win.ShowWindow(hwnd, win.SW_SHOW)
	win.ShowWindow(hwnd, win.SW_RESTORE)
	win.SetForegroundWindow(hwnd)
}
