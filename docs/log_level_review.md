# ログレベル見直し: 実装詳細

## 概要

このドキュメントは、Synology Office Exporterのログレベルの見直しと最適化について説明します。

## 問題点

従来のログ出力では、多くの詳細な処理情報が `info` レベルで出力されており、実運用時にログが冗長になっていました。

## 変更内容

### 1. デフォルトログレベルの変更
- **変更前**: `warn` (重要な情報が表示されない)
- **変更後**: `info` (適切なバランス)

### 2. 個別ログメッセージのレベル調整

#### Debug レベルに変更したメッセージ
以下のメッセージを `info` から `debug` に変更しました：

**export_processor.go**:
- `"Skipping non-exportable file"` - 各ファイルの詳細処理情報
- `"Dry run: would export file"` - ドライラン時の詳細情報
- `"Exporting file"` - 各ファイルの詳細処理情報
- `"File exported successfully"` - 各ファイルの詳細処理情報

**file_operations.go**:
- `"Dry run: would remove file"` - ドライラン時の詳細情報
- `"File already removed"` - 詳細な状態情報
- `"File removed successfully"` - 詳細な処理情報

### 3. ログレベル使用指針

| レベル | 用途 | 例 |
|--------|------|-----|
| **Debug** | 詳細な処理情報、個別ファイル操作 | ファイル処理、ドライラン詳細 |
| **Info** | 重要な動作情報、統計 | 開始/完了メッセージ、統計情報 |
| **Warn** | 非致命的な警告 | 履歴更新失敗、リトライ可能エラー |
| **Error** | エラー状況 | ファイル処理失敗、システムエラー |

## 運用への影響

### デフォルト設定 (info レベル)
実行時に表示される情報:
- アプリケーション開始/完了
- エクスポート統計情報
- 重要な処理決定
- エラーと警告

表示されない情報:
- 個別ファイルの処理詳細
- ドライラン時の詳細操作

### Debug レベル使用時
すべての処理詳細が表示され、トラブルシューティングに有用です。

## 設定方法

```bash
# 標準的な使用 (info レベル)
export LOG_LEVEL=info
./synology-office-exporter

# 詳細デバッグ情報が必要な場合
export LOG_LEVEL=debug
./synology-office-exporter

# 最小限のログ (warn レベル)
export LOG_LEVEL=warn
./synology-office-exporter
```

## 変更されたファイル

1. `logger/config_loader.go` - デフォルトレベルを `warn` → `info` に変更
2. `synology_drive_exporter/export_processor.go` - 4つのメッセージを `info` → `debug` に変更
3. `synology_drive_exporter/file_operations.go` - 3つのメッセージを `info` → `debug` に変更
4. `README.md` - ログ設定のドキュメント更新

## 後方互換性

- 環境変数 `LOG_LEVEL` で明示的にレベルを指定している場合、動作は変更されません
- コマンドラインフラグ `-log-level` で指定している場合、動作は変更されません
- デフォルト動作のみが変更されています

## テスト

すべての既存テストが正常に通過することを確認済みです。ログレベルの変更はログ出力のみに影響し、アプリケーションの機能には影響しません。
