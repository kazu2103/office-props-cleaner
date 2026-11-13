// 指定した Office ファイルからプロパティを消すコマンド。
package main

import (
	"fmt"
	"os"

	"github.com/kazu2103/office-props-cleaner/core"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: scrub <file>...")
		os.Exit(2)
	}
	for _, path := range os.Args[1:] {
		core.ScrubAndLog(path, func(m string) { fmt.Println(m) })
	}
}
