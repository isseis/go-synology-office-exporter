# download_history パッケージ

Synology Drive Office Exporter のダウンロード履歴管理を担当する独立パッケージです。

- 履歴ファイルのロード・保存
- ダウンロードアイテムの状態管理
- 統計情報の集計

## 利用例
```go
import "github.com/isseis/go-synology-office-exporter/download_history"

history, err := download_history.NewDownloadHistory("history.json")
// ...
```
