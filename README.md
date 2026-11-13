# Officeプロパティ消去

指定したフォルダ（初期値はダウンロードフォルダ）を見張り、置かれた Excel / Word / PowerPoint ファイルから
作成者・最終更新者・会社名・管理者・ユーザー設定のプロパティと、Excel のブック保存場所を消す。
シートや本文、コメント、変更履歴には触らない。

## 使う人向け

- Mac はメニューバー、Windows はタスクトレイにアイコンが出る。そこから見張りの開始・停止、フォルダの変更、
  ファイルを選んでその場で消す、ログイン時の自動起動を操作する
- 手動で開いたときは止まった状態で起動し、「見張りを始める」を押すと始まる。ログイン時の自動起動では、すぐに見張りを始める
- 見張りを始めた時点でフォルダにあるファイルも対象になる
- 署名していないので、初回は Mac では右クリックから「開く」、Windows では SmartScreen の「詳細情報 > 実行」が要る

## 構成

```
core/      プロパティを消す処理・フォルダの見張り・設定・自動起動（Go。Mac / Windows 共通）
mac/       メニューバーの画面（Swift）と、core を呼ぶ橋渡し（Go の c-archive）
windows/   タスクトレイの画面（Go）
cmd/scrub  ファイルを指定して消すコマンド
```

## ビルド

- Mac: `mac/build.sh` → `dist/mac/OfficePropsCleaner.app`（ビルドした Mac の CPU 向けのみ）
- Windows: `windows/build.sh`（Mac からでもビルドできる）→ `dist/windows/OfficePropsCleaner.exe`

## 確認

```
go test ./core/
```
