# SQLC Use Analysis Demo

このデモは、SQLCプラグインの依存関係分析機能を実際に体験できるプログラムです。

## 概要

サンプルのe-commerceプロジェクト（ユーザー、投稿、コメント機能）を使用して、以下の分析機能をデモンストレーションします：

- **SQL クエリ分析**: テーブルアクセスパターンの抽出
- **Go コード分析**: 関数とメソッド呼び出しの解析
- **依存関係マッピング**: SQLクエリとGoコードの関係性の特定
- **レポート生成**: 複数形式での結果出力

## デモの種類

### 1. 基本デモ (cmd/demo)

自動実行されるデモで、分析結果をコンソールに表示し、JSONファイルに詳細を出力します。

**実行方法:**
```bash
go run cmd/demo/main.go
```

**出力例:**
```
=== SQLC Use Analysis Demo ===

Analyzing sample e-commerce project...
Project location: /path/to/test/fixtures/simple_project

1. Setting up analysis...
  • SQL Queries: 8
  • Go Packages: 3

2. Running dependency analysis...
  • Analysis completed in 45.2ms

3. Analysis Results
  • Functions analyzed: 27
  • Tables identified: 3
  • Dependencies found: 10

  Tables:
    • users (accessed by 5 functions)
    • posts (accessed by 3 functions)
    • comments (accessed by 2 functions)

  Operations:
    • SELECT: 6 times
    • INSERT: 4 times

4. Dependency Analysis
  Service Layer Analysis:
    • UserService.GetUser:
      - users: [SELECT] (1 calls)
    • UserService.CreateUser:
      - users: [INSERT] (1 calls)
    • PostService.GetPost:
      - posts: [SELECT] (1 calls)
      - users: [SELECT] (1 calls)

  Complex Dependencies:
    • PostService.GetPost accesses: [posts, users]
    • PostService.GetPostComments accesses: [comments, users]

5. Saving detailed results...
  • Results saved to demo_results.json (2847 bytes)

Demo completed successfully!
Check './demo_results.json' for detailed analysis results.
```

### 2. インタラクティブデモ (cmd/interactive-demo)

メニュー形式で様々な分析機能を体験できるデモです。

**実行方法:**
```bash
go run cmd/interactive-demo/main.go
```

**機能:**
- 基本分析の実行
- プロジェクト構造の表示
- SQLクエリ一覧の表示
- 依存関係グラフの表示
- テーブル別分析
- 関数別分析
- 結果のエクスポート
- エラー分析

## デモプロジェクト構造

デモで使用するサンプルプロジェクト:

```
test/fixtures/simple_project/
├── schema.sql              # データベーススキーマ
├── query.sql              # SQLクエリ定義
├── sqlc.yaml              # SQLC設定
└── internal/
    ├── db/                # データベース層
    │   ├── models.go      # モデル定義
    │   └── query.sql.go   # SQLC生成コード
    ├── service/           # ビジネスロジック層
    │   ├── user_service.go
    │   └── post_service.go
    └── handler/           # HTTPハンドラー層
        ├── user_handler.go
        └── post_handler.go
```

## 分析対象

### データベーステーブル
- **users**: ユーザー情報 (id, name, email, created_at)
- **posts**: 投稿情報 (id, title, content, author_id, created_at)
- **comments**: コメント情報 (id, post_id, author_id, content, created_at)

### SQLクエリ
1. **GetUser**: ユーザー取得
2. **ListUsers**: ユーザー一覧
3. **CreateUser**: ユーザー作成
4. **GetPost**: 投稿取得（JOINあり）
5. **ListPostsByUser**: ユーザー別投稿一覧
6. **CreatePost**: 投稿作成
7. **GetCommentsByPost**: 投稿別コメント一覧（JOINあり）
8. **CreateComment**: コメント作成

### Go関数
- **データベース層**: SQLC生成の27関数
- **サービス層**: 8つのビジネスロジック関数
- **ハンドラー層**: 6つのHTTPハンドラー関数

## 分析結果の例

### 依存関係マッピング
```json
{
  "functions": {
    "UserService.GetUser": {
      "name": "GetUser",
      "package": "service",
      "table_access": {
        "users": {
          "operations": ["SELECT"],
          "methods": ["GetUser"],
          "count": 1
        }
      }
    }
  },
  "dependencies": [
    {
      "function": "UserService.GetUser",
      "table": "users",
      "operation": "SELECT",
      "method": "GetUser",
      "line": 31
    }
  ]
}
```

### 複雑な依存関係
JOINクエリを含む関数では複数テーブルへのアクセスが検出されます：

- `PostService.GetPost` → `posts`, `users`
- `PostService.GetPostComments` → `comments`, `users`

## エラーハンドリングのデモ

デモでは強化されたエラーハンドリングシステムも体験できます：

- **パニック回復**: 予期しないエラーからの自動回復
- **部分失敗処理**: 一部の分析が失敗しても処理続行
- **構造化ログ**: 詳細なエラー情報の記録
- **ユーザーフレンドリーメッセージ**: 理解しやすいエラー説明

## 出力形式

### JSON形式
```json
{
  "functions": {...},
  "tables": {...},
  "dependencies": [...],
  "summary": {
    "function_count": 27,
    "table_count": 3,
    "dependency_count": 10,
    "operation_counts": {
      "SELECT": 6,
      "INSERT": 4
    }
  }
}
```

### テキスト形式
```
SQLC Use Analysis Report
========================

Summary:
- Functions: 27
- Tables: 3
- Dependencies: 10

Tables:
- users (accessed by 5 functions)
- posts (accessed by 3 functions)
- comments (accessed by 2 functions)

Dependencies:
- UserService.GetUser -> users (SELECT via GetUser)
- UserService.CreateUser -> users (INSERT via CreateUser)
...
```

## 前提条件

- Go 1.21以上
- プロジェクトルートディレクトリからの実行

## トラブルシューティング

### "Demo fixture not found" エラー
プロジェクトのルートディレクトリから実行してください：
```bash
cd /path/to/sqlc-use-analysis
go run cmd/demo/main.go
```

### 分析エラー
インタラクティブデモの「Show Error Analysis」メニューでエラー詳細を確認できます。

## 次のステップ

デモ実行後は以下を試してみてください：

1. **カスタムクエリの追加**: 独自のSQLクエリでの分析
2. **複雑なプロジェクトでの実行**: 実際のプロジェクトでの使用
3. **出力形式のカスタマイズ**: JSON/CSV/HTMLでの結果出力
4. **エラーハンドリングの確認**: 意図的にエラーを発生させての動作確認

## フィードバック

デモに関するフィードバックは以下まで：
- GitHub Issues: https://github.com/anthropics/claude-code/issues
- 改善提案やバグ報告をお待ちしています