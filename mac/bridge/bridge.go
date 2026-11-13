// Swift の画面から core を呼ぶための C 関数。go build -buildmode=c-archive で静的ライブラリにする。
package main

/*
#include <stdlib.h>
#include <stdbool.h>
*/
import "C"

import (
	"encoding/json"
	"sync"

	"github.com/kazu2103/office-props-cleaner/core"
)

var (
	logs = make(chan string, 1000)
	mu   sync.Mutex
	stop chan struct{}
)

func log(message string) {
	select {
	case logs <- message:
	default:
	}
}

//export NextLog
func NextLog() *C.char {
	select {
	case m := <-logs:
		return C.CString(m)
	default:
		return nil
	}
}

//export StartWatch
func StartWatch(folder *C.char, recursive C.bool) {
	mu.Lock()
	defer mu.Unlock()
	if stop != nil {
		close(stop)
	}
	stop = make(chan struct{})
	go core.Watch(C.GoString(folder), bool(recursive), log, stop)
}

//export StopWatch
func StopWatch() {
	mu.Lock()
	defer mu.Unlock()
	if stop != nil {
		close(stop)
		stop = nil
	}
}

//export ScrubPath
func ScrubPath(path *C.char) {
	core.ScrubAndLog(C.GoString(path), log)
}

//export IsWatchOnLaunchArg
func IsWatchOnLaunchArg(arg *C.char) C.bool {
	return C.bool(C.GoString(arg) == core.WatchOnLaunchArg)
}

//export LoadSettingsJSON
func LoadSettingsJSON() *C.char {
	data, _ := json.Marshal(core.LoadSettings())
	return C.CString(string(data))
}

//export ValidateFolder
func ValidateFolder(folder *C.char) *C.char {
	if err := core.ValidateFolder(C.GoString(folder)); err != nil {
		return C.CString(err.Error())
	}
	return nil
}

//export SaveSettings
func SaveSettings(folder *C.char, autostart C.bool, recursive C.bool) *C.char {
	s := core.Settings{Folder: C.GoString(folder), Autostart: bool(autostart), Recursive: bool(recursive)}
	if err := core.SaveSettings(s); err != nil {
		return C.CString(err.Error())
	}
	if err := core.SetAutostart(s.Autostart); err != nil {
		return C.CString(err.Error())
	}
	return nil
}

func main() {}
