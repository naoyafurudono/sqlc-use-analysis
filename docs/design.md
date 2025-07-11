# sqlc依存関係解析プラグイン 設計書

## 1. システムアーキテクチャ

### 1.1. 全体構成

```
┌─────────────────────────────────────────────────────────────┐
│                     sqlc generate                           │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                  プラグインメインプロセス                      │
│  ┌─────────────────────┐ ┌──────────────────────┐          │
│  │  設定ローダー         │ │  エラーハンドラー      │          │
│  └─────────────────────┘ └──────────────────────┘          │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │               解析オーケストレーター                    │   │
│  └─────────────────────────────────────────────────────┘   │
│                   │                    │                    │
│  ┌────────────────▼──────┐ ┌─────────▼──────────┐         │
│  │  SQLクエリアナライザー   │ │  Go静的アナライザー  │         │
│  └───────────────────────┘ └────────────────────┘         │
│                   │                    │                    │
│  ┌────────────────▼────────────────────▼──────────┐        │
│  │            依存関係マッパー                      │        │
│  └────────────────────────┬───────────────────────┘        │
│                           │                                 │
│  ┌────────────────────────▼───────────────────────┐        │
│  │              JSON出力フォーマッター               │        │
│  └─────────────────────────────────────────────────┘        │
└─────────────────────────────────────────────────────────────┘
```

### 1.2. コンポーネント詳細

#### 1.2.1. プラグインメインプロセス (`main.go`)
- sqlcプラグインのエントリーポイント
- 標準入力から`CodeGeneratorRequest`を受信
- 各コンポーネントの初期化と実行管理

#### 1.2.2. 設定ローダー (`config/loader.go`)
- sqlc.yamlからプラグイン設定を読み込み
- 設定の検証とデフォルト値の適用

#### 1.2.3. SQLクエリアナライザー (`analyzer/sql_analyzer.go`)
- sqlcの`CodeGeneratorRequest`からクエリ情報を解析
- メソッド名、テーブル名、操作種別のマッピング生成

#### 1.2.4. Go静的アナライザー (`analyzer/go_analyzer.go`)
- `go/parser`、`go/types`を使用したソースコード解析
- 関数定義とメソッド呼び出しの追跡

#### 1.2.5. 依存関係マッパー (`mapper/dependency_mapper.go`)
- SQL解析結果とGo解析結果の結合
- 関数→テーブル→操作の依存関係ツリー構築

#### 1.2.6. JSON出力フォーマッター (`output/json_formatter.go`)
- 解析結果のJSON形式への変換
- function_viewとtable_viewの生成

## 2. データ構造設計

### 2.1. 内部データ構造

```go
// SQL解析結果
type SQLMethod struct {
    MethodName string
    TableName  string
    Operations []Operation // SELECT, INSERT, UPDATE, DELETE
}

// Go関数情報
type GoFunction struct {
    PackagePath string
    FunctionName string
    FilePath    string
    Line        int
    CalledMethods []string // sqlc生成メソッドの呼び出し
}

// 依存関係
type Dependency struct {
    Function   string
    Table      string
    Operations []string
}

// 解析結果
type AnalysisResult struct {
    FunctionView map[string][]TableOperation
    TableView    map[string][]FunctionOperation
}
```

### 2.2. 設定構造

```go
type Config struct {
    RootPath   string   // 解析対象のルートパス
    OutputPath string   // 出力ファイルパス
    Exclude    []string // 除外パターン
}
```

## 3. 処理フロー設計

### 3.1. メイン処理フロー

```
1. プラグイン起動
   └→ 標準入力からCodeGeneratorRequestを読み込み
   
2. 設定読み込み
   └→ sqlc.yamlからプラグイン設定を取得
   
3. SQL解析
   └→ クエリ定義からメソッド名とテーブル操作のマッピング作成
   
4. Go静的解析
   ├→ プロジェクトルートからGoファイルを収集
   ├→ 除外パターンに基づくフィルタリング
   └→ 各ファイルのAST解析と型情報の収集
   
5. 依存関係マッピング
   ├→ Go関数とsqlcメソッドの呼び出し関係を特定
   └→ 関数→テーブル→操作の依存関係を構築
   
6. 結果出力
   └→ JSON形式でファイルに出力
```

### 3.2. Go静的解析の詳細フロー

```go
func analyzeGoCode(rootPath string, exclude []string) ([]GoFunction, error) {
    // 1. ファイル収集
    files := collectGoFiles(rootPath, exclude)
    
    // 2. パッケージの型情報ロード
    cfg := &packages.Config{
        Mode: packages.NeedName | packages.NeedFiles | 
              packages.NeedImports | packages.NeedTypes |
              packages.NeedSyntax | packages.NeedTypesInfo,
    }
    pkgs, err := packages.Load(cfg, rootPath)
    
    // 3. 各パッケージの解析
    for _, pkg := range pkgs {
        // AST走査
        for _, file := range pkg.Syntax {
            ast.Inspect(file, func(n ast.Node) bool {
                // 関数定義の検出
                // メソッド呼び出しの追跡
                return true
            })
        }
    }
}
```

## 4. エラーハンドリング設計

### 4.1. エラー種別

1. **設定エラー**
   - 必須パラメータの欠如
   - 無効なパス指定

2. **解析エラー**
   - Goコードのパースエラー
   - SQL文の解析エラー

3. **実行時エラー**
   - ファイルI/Oエラー
   - メモリ不足

### 4.2. エラー処理方針

```go
type PluginError struct {
    Type    ErrorType
    Message string
    Details map[string]interface{}
}

// エラーは標準エラー出力に構造化ログとして出力
// sqlcがエラーを認識できるように適切な終了コードを返す
```

## 5. パフォーマンス最適化

### 5.1. 並行処理
- Goファイルの解析を並行実行
- ワーカープールパターンの採用

### 5.2. メモリ効率
- 大規模プロジェクトでのストリーミング処理
- 不要なAST情報の早期解放

### 5.3. キャッシング
- 型情報のキャッシュによる再利用
- 前回の解析結果との差分検出（将来的な拡張）

## 6. テスト戦略

### 6.1. ユニットテスト
- 各コンポーネントの独立したテスト
- モックを使用した依存関係の分離

### 6.2. 統合テスト
- サンプルプロジェクトを使用したE2Eテスト
- 様々なコードパターンでの動作確認

### 6.3. パフォーマンステスト
- 大規模プロジェクトでの実行時間測定
- メモリ使用量のプロファイリング

## 7. 拡張性考慮

### 7.1. プラグインアーキテクチャ
- 新しい解析ルールの追加が容易な設計
- 出力フォーマットの拡張（CSV、HTML等）

### 7.2. 設定の柔軟性
- 解析深度の調整
- カスタムフィルタリングルール

## 8. セキュリティ考慮

### 8.1. 入力検証
- パス・トラバーサル攻撃の防止
- 設定値のサニタイゼーション

### 8.2. リソース制限
- 解析対象ファイルサイズの上限設定
- 実行時間のタイムアウト設定