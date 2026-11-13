"""ダウンロードフォルダに置かれた Office ファイル（Excel / Word / PowerPoint）から、
作成者・最終更新者・会社名などのプロパティと、Excel のブック保存場所を消す。

使い方:
  python3 office_props_cleaner.py             # ~/Downloads を見張り続ける
  python3 office_props_cleaner.py <フォルダ>   # 見張るフォルダを指定
  python3 office_props_cleaner.py --once <ファイル>...  # 指定ファイルだけ処理
  python3 office_props_cleaner.py --selftest
"""

import os
import re
import sys
import time
import zipfile
from pathlib import Path

TARGET_SUFFIXES = {".xlsx", ".xlsm", ".docx", ".docm", ".pptx", ".pptm"}
POLL_SECONDS = 2

EMPTY_CORE = (
    b'<?xml version="1.0" encoding="UTF-8" standalone="yes"?>\r\n'
    b'<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties"'
    b' xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/"'
    b' xmlns:dcmitype="http://purl.org/dc/dcmitype/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"/>'
)
APP_PERSONAL = re.compile(rb"<(Company|Manager)>[^<]*</\1>|<(Company|Manager)\s*/>")
CUSTOM_PROPERTY = re.compile(rb"<property\b.*?</property>", re.S)
WORKBOOK_ABS_PATH = re.compile(
    rb"<mc:AlternateContent(?:(?!</mc:AlternateContent>).)*?absPath(?:(?!</mc:AlternateContent>).)*?</mc:AlternateContent>",
    re.S,
)


def scrub_part(name, data):
    if name == "docProps/core.xml":
        return EMPTY_CORE
    if name == "docProps/app.xml":
        return APP_PERSONAL.sub(b"", data)
    if name == "docProps/custom.xml":
        return CUSTOM_PROPERTY.sub(b"", data)
    if name == "xl/workbook.xml":
        return WORKBOOK_ABS_PATH.sub(b"", data)
    return data


def scrub_file(path: Path) -> bool:
    with zipfile.ZipFile(path) as src:
        entries = [(info, src.read(info.filename)) for info in src.infolist()]
    scrubbed = [(info, scrub_part(info.filename, data)) for info, data in entries]
    if all(new == old for (_, new), (_, old) in zip(scrubbed, entries)):
        return False

    tmp = path.with_name(f".{path.name}.scrubbing")
    with zipfile.ZipFile(tmp, "w") as dst:
        for info, data in scrubbed:
            dst.writestr(info, data, compress_type=info.compress_type)
    os.replace(tmp, path)
    return True


def is_target(path: Path) -> bool:
    return path.suffix.lower() in TARGET_SUFFIXES and not path.name.startswith((".", "~$"))


def log_print(message):
    print(message, flush=True)


def try_scrub(path: Path, log=log_print):
    try:
        if scrub_file(path):
            log(f"消しました: {path.name}")
    except (zipfile.BadZipFile, PermissionError, FileNotFoundError, OSError) as e:
        # ダウンロード途中・Excel で開いている最中などは次の周回でやり直す
        log(f"スキップ: {path.name} ({e.__class__.__name__})")


def watch(folder: Path, log=log_print, should_stop=lambda: False):
    log(f"見張り中: {folder}")
    last_seen = {}
    done = {}
    while not should_stop():
        for path in folder.iterdir():
            if not path.is_file() or not is_target(path):
                continue
            signature = file_signature(path)
            if done.get(path) == signature:
                continue
            # 書き込み中のファイルを触らないよう、サイズと更新時刻が1周回変わらなかったものだけ処理する
            if last_seen.get(path) == signature:
                try_scrub(path, log)
                done[path] = file_signature(path)
            last_seen[path] = signature
        time.sleep(POLL_SECONDS)


def file_signature(path: Path):
    stat = path.stat()
    return stat.st_size, stat.st_mtime


def selftest():
    import io

    buf = io.BytesIO()
    with zipfile.ZipFile(buf, "w", zipfile.ZIP_DEFLATED) as z:
        z.writestr("docProps/core.xml", "<cp:coreProperties><dc:creator>他社 太郎</dc:creator></cp:coreProperties>")
        z.writestr("docProps/app.xml", "<Properties><Company>他社株式会社</Company><Manager/></Properties>")
        z.writestr("xl/workbook.xml", '<workbook><mc:AlternateContent><mc:Choice><x15ac:absPath url="C:\\Users\\taro\\"/></mc:Choice></mc:AlternateContent><sheets/></workbook>')
        z.writestr("xl/worksheets/sheet1.xml", "<worksheet>本文の他社</worksheet>")
    path = Path(os.environ.get("TMPDIR", "/tmp")) / "selftest.xlsx"
    path.write_bytes(buf.getvalue())

    assert scrub_file(path) is True
    with zipfile.ZipFile(path) as z:
        props = z.read("docProps/core.xml") + z.read("docProps/app.xml") + z.read("xl/workbook.xml")
        assert "他社".encode() not in props and b"absPath" not in props
        assert b"<sheets/>" in z.read("xl/workbook.xml")
        assert z.read("xl/worksheets/sheet1.xml") == "<worksheet>本文の他社</worksheet>".encode()
    assert scrub_file(path) is False
    path.unlink()
    print("selftest ok")


if __name__ == "__main__":
    args = sys.argv[1:]
    if args == ["--selftest"]:
        selftest()
    elif args[:1] == ["--once"]:
        for p in args[1:]:
            try_scrub(Path(p))
    else:
        watch(Path(args[0]) if args else Path.home() / "Downloads")
