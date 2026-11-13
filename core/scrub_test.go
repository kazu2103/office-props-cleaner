package core

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func readEntry(t *testing.T, path, name string) string {
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name == name {
			rc, _ := f.Open()
			defer rc.Close()
			b, _ := io.ReadAll(rc)
			return string(b)
		}
	}
	t.Fatalf("%s がない", name)
	return ""
}

func TestScrubFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.xlsx")
	out, _ := os.Create(path)
	w := zip.NewWriter(out)
	for name, body := range map[string]string{
		"docProps/core.xml":        `<cp:coreProperties><dc:creator>他社 太郎</dc:creator></cp:coreProperties>`,
		"docProps/app.xml":         `<Properties><Company>他社株式会社</Company><Manager/></Properties>`,
		"xl/workbook.xml":          `<workbook><mc:AlternateContent xmlns:mc="m"><mc:Choice Requires="x15"><x15ac:absPath url="C:\Users\taro\" xmlns:x15ac="x"/></mc:Choice></mc:AlternateContent><sheets/></workbook>`,
		"xl/worksheets/sheet1.xml": `<worksheet>本文の他社</worksheet>`,
	} {
		fw, _ := w.Create(name)
		fw.Write([]byte(body))
	}
	w.Close()
	out.Close()

	modTime := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	os.Chtimes(path, modTime, modTime)

	changed, err := ScrubFile(path)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	props := readEntry(t, path, "docProps/core.xml") + readEntry(t, path, "docProps/app.xml") + readEntry(t, path, "xl/workbook.xml")
	if strings.Contains(props, "他社") || strings.Contains(props, "absPath") {
		t.Errorf("名前か保存場所が残っている: %s", props)
	}
	if got := readEntry(t, path, "xl/workbook.xml"); got != "<workbook><sheets/></workbook>" {
		t.Errorf("workbook.xml = %s", got)
	}
	if got := readEntry(t, path, "xl/worksheets/sheet1.xml"); got != "<worksheet>本文の他社</worksheet>" {
		t.Errorf("本文が変わった: %s", got)
	}
	if info, _ := os.Stat(path); !info.ModTime().Equal(modTime) {
		t.Errorf("更新日時が変わった: %v", info.ModTime())
	}
	if changed, _ := ScrubFile(path); changed {
		t.Error("2回目は書き換えないはず")
	}
}

func TestIsTarget(t *testing.T) {
	for path, want := range map[string]bool{
		"a.xlsx": true, "b.DOCX": true, "c.pptm": true, "d.pdf": false, "~$a.xlsx": false, ".a.xlsx.scrubbing": false,
	} {
		if IsTarget(path) != want {
			t.Errorf("IsTarget(%s) != %v", path, want)
		}
	}
}
