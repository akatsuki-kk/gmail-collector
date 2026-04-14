# gmail-collector

Gmail を検索し、メール本文から正規表現で値を抽出して JSON を出力する Go 製 CLI です。

## 必要な準備

1. Google Cloud でプロジェクトを作成する
2. Gmail API を有効化する
3. OAuth 同意画面を設定する
4. OAuth Client ID を作成する
5. Redirect URI に `http://127.0.0.1:8787/callback` を登録する

初回実行時は CLI が対話型で `Client ID` と `Client Secret` を確認し、ローカルユーザーファイルに保存します。以後は保存済みファイルを使って認証します。

保存先の既定値:

- macOS: `~/Library/Application Support/gmail-collector/`
- Linux: `~/.config/gmail-collector/`

保存されるファイル:

- `credentials.json`
- `token.json`

## インストール

```bash
go mod tidy
```

## ビルド

```bash
go build -o gmail-collector .
```

## 設定ファイル

設定は 1 つの YAML にまとめます。

```yaml
search:
  from:
    - "sender@example.com"
  subject_contains:
    - "お客様のID"
  body_contains:
    - "本人確認コード"
  after: "2025/01/01"
  before: "2025/12/31"
  label:
    - "inbox"
  include_spam_trash: false

extract:
  id: 'お客様のIDは（(.+?)）'
  phone_number: '電話番号は ([0-9-]+)'

output:
  pretty: true
```

`search` では構造化した条件を指定し、内部で Gmail の検索クエリに変換します。

- `from`: 送信者メールアドレスの配列
- `subject_contains`: 件名に含まれる文字列の配列
- `body_contains`: 本文に含まれている必要がある文字列の配列
- `after`: 開始日。Gmail クエリへそのまま渡す
- `before`: 終了日。Gmail クエリへそのまま渡す
- `label`: Gmail ラベル名の配列
- `include_spam_trash`: 迷惑メールとゴミ箱を検索対象に含めるか

`body_contains` は Gmail の検索クエリにも反映しつつ、取得後の本文テキストに対して再確認します。

`extract` は `キー: 正規表現` の形式です。正規表現にキャプチャグループがある場合は最初のグループを採用し、ない場合は一致全体を採用します。同じキーで複数一致した場合は最初の一致だけを返します。

HTML メールは本文をテキスト化してから抽出します。`text/plain` がある場合はそちらを優先します。

## 実行

```bash
./gmail-collector run --config config.example.yaml
```

ファイルへ出力する場合:

```bash
./gmail-collector run --config config.example.yaml --output result.json
```

実行時はまず検索対象件数を表示し、その後に各メールの処理進捗と抽出結果件数を表示してから JSON を出力します。

出力例:

```json
[
  {
    "message_id": "18c1d2example",
    "thread_id": "18c1d2thread",
    "subject": "お客様のIDのお知らせ",
    "from": "sender@example.com",
    "date": "2025-03-01T09:00:00+09:00",
    "extracted": {
      "id": "ABC123",
      "phone_number": "000-0000-9999"
    }
  }
]
```

## テスト

```bash
source ~/.zsh/.zshrc
workspec ./...
```
