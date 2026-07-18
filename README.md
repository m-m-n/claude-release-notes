# claude-release-notes

`anthropics/claude-code` の GitHub リリースを定期的に取得し、日本語化して
Gmail 経由でメール配信する単発実行の CLI ツール。スケジューリングは
systemd のユーザータイマーに委譲する。

## ビルドとインストール

deb パッケージを生成してインストールする。

```sh
make dpkg
sudo dpkg -i build/claude-release-notes_0.1.0_amd64.deb
```

バイナリは `/usr/bin/claude-release-notes` にインストールされる。

開発時のビルドのみ行う場合:

```sh
go build ./...
```

## 設定

設定ファイルは `$XDG_CONFIG_HOME/claude-release-notes/config.yaml`
（`XDG_CONFIG_HOME` 未設定時は `~/.config/claude-release-notes/config.yaml`）
に配置する。

### 設定キー一覧

| キー | 説明 |
|------|------|
| `gmail.account` | 送信元 Gmail アドレス |
| `gmail.app_password` | Gmail アプリパスワード |
| `mail.to` | 宛先メールアドレス |
| `github.token` | GitHub Personal Access Token |

### 設定ファイル例（値は伏せ字）

```yaml
gmail:
  account: your-account@gmail.com
  app_password: "xxxx xxxx xxxx xxxx"
mail:
  to: recipient@example.com
github:
  token: github_pat_xxxxxxxxxxxxxxxxxxxx
```

設定ファイルには認証情報が含まれるため、所有者のみ読み書き可能に制限する。

```sh
chmod 600 ~/.config/claude-release-notes/config.yaml
```

### Gmail アプリパスワードの取得

Google アカウントの「アプリ パスワード」ページから取得する。

https://myaccount.google.com/apppasswords

### GitHub トークンの取得

`github.token` には GitHub Personal Access Token（Fine-grained token）を
設定する。`anthropics/claude-code` の Releases API 呼び出しに使用する。

1. Fine-grained token の発行ページを開く。

   https://github.com/settings/personal-access-tokens/new

2. 以下を設定して「Generate token」を押す。

   | 項目 | 設定値 |
   |------|--------|
   | Token name | 任意（例: `claude-release-notes`） |
   | Expiration | 任意の有効期限 |
   | Repository access | **Public repositories (read-only)** |
   | Permissions | 追加不要 |

3. 表示されたトークン（`github_pat_` で始まる）を `config.yaml` の
   `github.token` に設定する。トークンはこの画面でしか表示されない。

有効期限が切れると取得が失敗するようになるため、失効時は再発行して
`config.yaml` を更新する。

## 状態ファイル

取得済み最新バージョンは
`$XDG_STATE_HOME/claude-release-notes/state.json`
（`XDG_STATE_HOME` 未設定時は `~/.local/state/claude-release-notes/state.json`）
に保存される。

## systemd タイマーのインストール

1. deb パッケージでバイナリをインストールしておく。
   `systemd/claude-release-notes.service` の `ExecStart` は
   `/usr/bin/claude-release-notes` を指す。別の場所に配置した場合は
   書き換える。
2. unit ファイルをユーザー unit ディレクトリにコピーする。

   ```sh
   mkdir -p ~/.config/systemd/user
   cp systemd/claude-release-notes.service systemd/claude-release-notes.timer \
     ~/.config/systemd/user/
   ```

3. systemd に変更を認識させる。

   ```sh
   systemctl --user daemon-reload
   ```

4. タイマーを有効化して起動する。

   ```sh
   systemctl --user enable --now claude-release-notes.timer
   ```

## 実行結果の確認

```sh
journalctl --user -u claude-release-notes.service
```

## 手動実行

タイマーを待たずに手動で実行する場合、インストール先のバイナリを直接実行する。

```sh
claude-release-notes
```
