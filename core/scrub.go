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
func ScrubFile(path string) (bool, error) {
	src, err := zip.OpenReader(path)
	if err != nil {
		return false, err
	}
	defer src.Close()

	type entry struct {
		header *zip.FileHeader
		data   []byte
	}
	var entries []entry
	changed := false
	for _, f := range src.File {
		rc, err := f.Open()
		if err != nil {
			return false, err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return false, err
		}
		scrubbed := scrubPart(f.Name, data)
		if !bytes.Equal(scrubbed, data) {
			changed = true
		}
		header := f.FileHeader
		entries = append(entries, entry{&header, scrubbed})
	}
	if !changed {
		return false, nil
	}

	tmp := filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".scrubbing")
	out, err := os.Create(tmp)
	if err != nil {
		return false, err
	}
	w := zip.NewWriter(out)
	for _, e := range entries {
		h := *e.header
		fw, err := w.CreateHeader(&h)
		if err == nil {
			_, err = fw.Write(e.data)
		}
		if err != nil {
			out.Close()
			os.Remove(tmp)
			return false, err
		}
	}
	if err := w.Close(); err != nil {
		out.Close()
		os.Remove(tmp)
		return false, err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return false, err
	}
	src.Close()
	return true, os.Rename(tmp, path)
}
