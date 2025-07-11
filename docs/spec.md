# sqlcプラグイン新規開発 要件定義書

  - **文書バージョン:** 1.0
  - **作成日:** 2025年7月11日
  - **作成者:** Gemini

## 1. 概要

### 1.1. 背景と課題

現在のGoプロジェクトにおいて、どのビジネスロジックがどのデータベーステーブルにアクセスしているかの依存関係が、コードを詳細に読み解かないと把握できない状態にある。
このため、以下の課題が生じている。

  - **影響範囲の調査コスト:** テーブルスキーマの変更や機能改修の際に、影響を受ける箇所を特定するのに時間がかかる。
  - **アーキテクチャの把握困難:** システム全体のデータフローが不明瞭で、新規参画者のオンボーディングやアーキテクチャレビューの障壁となっている。
  - **意図しない依存関係の発生:** 本来アクセスすべきでないコンポーネントからのDBアクセスなど、潜在的な問題を検知しにくい。

### 1.2. 目的とゴール

本プラグインは、Goのソースコードを静的解析し、**「どの関数」が「どのテーブル」に「どのSQL操作（`SELECT`, `INSERT`, `UPDATE`, `DELETE`）」を行っているか**を可視化することを目的とする。

これにより、データベースアクセスの依存関係を明確にし、コードの保守性、開発生産性、およびシステムの信頼性を向上させることをゴールとする。

## 2. プラグインの仕様

### 2.1. 機能概要

  - **入力:** sqlcのクエリ定義ファイル (`.sql`) と、Goプロジェクトのソースコード。
  - **処理:** Goの静的解析を行い、関数単位でのテーブル操作（`SELECT`, `INSERT`, `UPDATE`, `DELETE`）を特定する。
  - **出力:** 解析結果をJSONファイルとして出力する。

### 2.2. 実行方法

本プラグインはsqlcのプラグインとして実装され、`sqlc generate` コマンド実行時に自動的に呼び出される。

## 3. 機能要件

### 3.1. 解析プロセス

プラグインは以下のステップで解析を実行する。

1.  **SQL解析:** sqlcから提供される `CodeGeneratorRequest` を元に、定義された各クエリを解析する。

      - 生成されるGoのメソッド名
      - 操作対象のテーブル名
      - SQL操作の種類（`SELECT`, `INSERT`, `UPDATE`, `DELETE`）

    上記3つの対応関係をメモリ上にマッピングする。

2.  **Goコード静的解析:** Goの標準ライブラリ（`go/parser`, `go/types`等）を用いて、設定されたGoプロジェクトのソースコード全体を解析する。関数定義と、その関数内で呼び出されているメソッドを走査する。

3.  **依存関係マッピング:** 上記1と2の結果を結合する。Goの関数がsqlcの生成メソッドを呼び出している箇所を特定し、「関数 → テーブル → 操作」の依存関係ツリーを構築する。

### 3.2. 出力仕様

解析結果はJSON形式で出力する。JSONは、`function_view` と `table_view` の2つの視点を持つオブジェクトとする。

  - **`function_view`:** 関数をキーとし、その関数がアクセスするテーブルと操作のリストを値とする。
  - **`table_view`:** テーブルをキーとし、そのテーブルにアクセスする関数と操作のリストを値とする。

#### JSONフォーマット例

```json
{
  "function_view": {
    "github.com/your_org/your_project/billing.CreateInvoice": [
      {
        "table": "invoices",
        "operations": ["INSERT"]
      },
      {
        "table": "customers",
        "operations": ["SELECT"]
      }
    ],
    "github.com/your_org/your_project/auth.RegisterUser": [
      {
        "table": "users",
        "operations": ["INSERT"]
      }
    ]
  },
  "table_view": {
    "invoices": [
      {
        "function": "github.com/your_org/your_project/billing.CreateInvoice",
        "operations": ["INSERT"]
      }
    ],
    "customers": [
      {
        "function": "github.com/your_org/your_project/billing.CreateInvoice",
        "operations": ["SELECT"]
      }
    ],
    "users": [
      {
        "function": "github.com/your_org/your_project/auth.RegisterUser",
        "operations": ["INSERT"]
      }
    ]
  }
}
```

## 4. 設定項目

プラグインの動作は `sqlc.yaml` ファイルで制御する。

#### `sqlc.yaml` 設定例

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "db/query/"
    schema: "db/migration/"
    gen:
      go:
        package: "db"
        out: "db/sqlc"
plugins:
  - name: "go_dependency_analyzer" # プラグイン名（任意）
    process:
      # プラグインの実行コマンド or Dockerイメージ
      cmd: "go-dependency-analyzer" 
    options:
      # プラグインに渡す設定
      root_path: "."
      output_path: "docs/db_dependencies.json"
      exclude:
        - "**/*_test.go"
        - "vendor/"
```

  - **`root_path` (string, required):** 解析対象となるGoプロジェクトのルートディレクトリパス。
  - **`output_path` (string, required):** 解析結果JSONの出力先ファイルパス。
  - **`exclude` (array of strings, optional):** 解析から除外するファイル/ディレクトリのパターンリスト。

## 5. 非機能要件

  - **開発言語:** Go
  - **パフォーマンス:** 大規模なコードベース（数万行〜数十万行）においても、数分以内に解析が完了することを目指す。
  - **エラーハンドリング:** GoのコードやSQLの構文にパース不可能な箇所があった場合、エラー箇所を特定できるメッセージを出力し、異常終了すること。
