// Package core は Office ファイル（Excel / Word / PowerPoint）から、作成者・最終更新者・会社名などの
// プロパティと Excel のブック保存場所を消す。Mac 版・Windows 版の両方から使う。
package core

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var targetSuffixes = map[string]bool{
	".xlsx": true, ".xlsm": true, ".docx": true, ".docm": true, ".pptx": true, ".pptm": true,
}

var emptyCore = []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\r\n" +
	`<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties"` +
	` xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/"` +
	` xmlns:dcmitype="http://purl.org/dc/dcmitype/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"/>`)

var (
	appPersonal     = regexp.MustCompile(`<(?:Company|Manager)>[^<]*</(?:Company|Manager)>|<(?:Company|Manager)\s*/>`)
	customProperty  = regexp.MustCompile(`(?s)<property\b.*?</property>`)
	workbookAbsPath = regexp.MustCompile(`<mc:AlternateContent\b[^>]*>\s*<mc:Choice\b[^>]*>\s*<x15ac:absPath\b[^>]*/>\s*</mc:Choice>\s*</mc:AlternateContent>`)
)

func IsTarget(path string) bool {
	name := filepath.Base(path)
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "~$") {
		return false
	}
	return targetSuffixes[strings.ToLower(filepath.Ext(name))]
}

func scrubPart(name string, data []byte) []byte {
	switch name {
	case "docProps/core.xml":
		return emptyCore
	case "docProps/app.xml":
		return appPersonal.ReplaceAll(data, nil)
	case "docProps/custom.xml":
		return customProperty.ReplaceAll(data, nil)
	case "xl/workbook.xml":
		return workbookAbsPath.ReplaceAll(data, nil)
	}
	return data
}

// ScrubFile は消すものがあったときだけファイルを書き換え、書き換えたかどうかを返す。
// 別ファイルと入れ替えると作成日時や「インターネットから取得した」目印（Mark of the Web / quarantine）が
// 失われ、Office の保護ビューが効かなくなるため、元のファイルをその場で書き換えて更新日時も戻す。
func ScrubFile(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	original, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	scrubbed, changed, err := scrubZip(original)
	if err != nil || !changed {
		return false, err
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return false, err
	}
	if _, err := f.Write(scrubbed); err != nil {
		f.Truncate(0)
		f.WriteAt(original, 0)
		f.Close()
		return false, err
	}
	if err := f.Close(); err != nil {
		return false, err
	}
	return true, os.Chtimes(path, time.Now(), info.ModTime())
}

func scrubZip(original []byte) ([]byte, bool, error) {
	src, err := zip.NewReader(bytes.NewReader(original), int64(len(original)))
	if err != nil {
		return nil, false, err
	}
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	changed := false
	for _, f := range src.File {
		rc, err := f.Open()
		if err != nil {
			return nil, false, err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, false, err
		}
		scrubbed := scrubPart(f.Name, data)
		if !bytes.Equal(scrubbed, data) {
			changed = true
		}
		header := f.FileHeader
		fw, err := w.CreateHeader(&header)
		if err != nil {
			return nil, false, err
		}
		if _, err := fw.Write(scrubbed); err != nil {
			return nil, false, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, false, err
	}
	return out.Bytes(), changed, nil
}
