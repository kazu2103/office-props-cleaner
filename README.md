# Officeプロパティ消去

指定したフォルダ（初期値はダウンロードフォルダ）を見張り、置かれた Excel / Word / PowerPoint ファイルから
作成者・最終更新者・会社名・管理者・ユーザー設定のプロパティと、Excel のブック保存場所を消す。
シートや本文、コメント、変更履歴には触らない。

## 使う人向け

1. アプリを開くと、すぐに見張りを始める（起動時点でフォルダにあるファイルも対象）
2. 「パソコンにログインしたら自動で見張りを始める」に印を付けると、次回から自動で動く
3. ウィンドウの閉じるボタンは最小化になる。止めるときは「終了」

## ビルド

PyInstaller はビルドした OS 向けのアプリしか作れないので、Mac と Windows でそれぞれビルドする。

```
uv run --no-project --python 3.13 --with pyinstaller pyinstaller --noconfirm --windowed --name "Officeプロパティ消去" app.py
```

- Mac: `dist/Officeプロパティ消去.app`
- Windows: `dist/Officeプロパティ消去/Officeプロパティ消去.exe`（`--onefile` を付けると1ファイルになる）

署名していないアプリは、初回起動時に Mac では「開発元を確認できない」、Windows では SmartScreen の警告が出る。

## 確認

```
python3 office_props_cleaner.py --selftest
```
