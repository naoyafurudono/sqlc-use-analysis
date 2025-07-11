# SQLクエリアナライザー 詳細設計

## 1. 概要

SQLクエリアナライザーは、sqlcから提供される`CodeGeneratorRequest`を解析し、各クエリから以下の情報を抽出します：
- 生成されるGoメソッド名
- 操作対象のテーブル名
- SQL操作の種類（SELECT, INSERT, UPDATE, DELETE）

## 2. 入力データ構造

### 2.1. sqlc CodeGeneratorRequest
```protobuf
message CodeGeneratorRequest {
  Settings settings = 1;
  Catalog catalog = 2;
  repeated Query queries = 3;
  string sqlc_version = 4;
}

message Query {
  string text = 1;             // SQL文
  string name = 2;             // クエリ名
  string cmd = 3;              // :one, :many, :exec, :execrows等
  repeated Column columns = 4;  // カラム情報
  repeated Parameter params = 5; // パラメータ情報
  string filename = 7;         // SQLファイル名
}
```

## 3. 解析ロジック

### 3.1. SQL操作種別の判定

```go
type OperationType string

const (
    OpSelect OperationType = "SELECT"
    OpInsert OperationType = "INSERT"
    OpUpdate OperationType = "UPDATE"
    OpDelete OperationType = "DELETE"
)

func detectOperationType(sqlText string) (OperationType, error) {
    // SQL文を正規化（改行、余分な空白を除去）
    normalizedSQL := normalizeSQLText(sqlText)
    
    // 最初のキーワードで判定
    upperSQL := strings.ToUpper(strings.TrimSpace(normalizedSQL))
    
    switch {
    case strings.HasPrefix(upperSQL, "SELECT"):
        return OpSelect, nil
    case strings.HasPrefix(upperSQL, "INSERT"):
        return OpInsert, nil
    case strings.HasPrefix(upperSQL, "UPDATE"):
        return OpUpdate, nil
    case strings.HasPrefix(upperSQL, "DELETE"):
        return OpDelete, nil
    case strings.HasPrefix(upperSQL, "WITH"):
        // CTE（Common Table Expression）の場合は本体を解析
        return detectCTEOperationType(upperSQL)
    default:
        return "", fmt.Errorf("unknown SQL operation: %s", sqlText)
    }
}
```

### 3.2. テーブル名の抽出

```go
type TableExtractor struct {
    parser *sqlparser.Parser
}

func (te *TableExtractor) ExtractTables(sqlText string) ([]string, error) {
    // SQL文のパース
    stmt, err := te.parser.Parse(sqlText)
    if err != nil {
        return nil, fmt.Errorf("failed to parse SQL: %w", err)
    }
    
    tables := make(map[string]bool)
    
    // ASTを走査してテーブル参照を収集
    ast.Walk(func(node ast.Node) bool {
        switch n := node.(type) {
        case *ast.TableName:
            tables[n.Name] = true
        case *ast.JoinClause:
            if n.Table != nil {
                tables[n.Table.Name] = true
            }
        }
        return true
    }, stmt)
    
    // 重複を除いてスライスに変換
    result := make([]string, 0, len(tables))
    for table := range tables {
        result = append(result, table)
    }
    
    return result, nil
}
```

### 3.3. メソッド名の生成規則

```go
func generateMethodName(queryName string, cmd string) string {
    // sqlcのメソッド名生成規則に準拠
    // クエリ名をPascalCaseに変換
    methodName := toPascalCase(queryName)
    
    // コマンドタイプに応じた調整
    switch cmd {
    case ":many":
        // 複数形にする場合の処理
        if !strings.HasSuffix(methodName, "s") &&
           !strings.HasSuffix(methodName, "List") {
            methodName = methodName + "s"
        }
    }
    
    return methodName
}
```

## 4. 複雑なSQLパターンへの対応

### 4.1. JOIN操作
```go
func extractTablesFromJoin(sqlText string) []TableInfo {
    // JOIN句を含むSQLの解析
    // 各JOINされたテーブルを抽出
    tables := []TableInfo{}
    
    // 正規表現でJOIN句を検出
    joinPattern := regexp.MustCompile(`(?i)(LEFT|RIGHT|INNER|FULL)?\s*JOIN\s+(\w+)`)
    matches := joinPattern.FindAllStringSubmatch(sqlText, -1)
    
    for _, match := range matches {
        tables = append(tables, TableInfo{
            Name: match[2],
            Type: "joined",
        })
    }
    
    return tables
}
```

### 4.2. サブクエリ
```go
func handleSubqueries(sqlText string) []QueryInfo {
    // サブクエリを再帰的に解析
    // 各サブクエリに対して操作種別とテーブルを特定
    subqueries := extractSubqueries(sqlText)
    
    results := []QueryInfo{}
    for _, subquery := range subqueries {
        opType, _ := detectOperationType(subquery)
        tables, _ := extractTables(subquery)
        
        results = append(results, QueryInfo{
            Operation: opType,
            Tables:    tables,
        })
    }
    
    return results
}
```

### 4.3. CTE（Common Table Expression）
```go
func parseCTE(sqlText string) ([]CTEInfo, string, error) {
    // WITH句の解析
    ctePattern := regexp.MustCompile(`(?i)WITH\s+(\w+)\s+AS\s*\((.*?)\)`)
    
    ctes := []CTEInfo{}
    mainQuery := sqlText
    
    // CTE定義を抽出
    matches := ctePattern.FindAllStringSubmatch(sqlText, -1)
    for _, match := range matches {
        cteName := match[1]
        cteQuery := match[2]
        
        opType, _ := detectOperationType(cteQuery)
        tables, _ := extractTables(cteQuery)
        
        ctes = append(ctes, CTEInfo{
            Name:      cteName,
            Operation: opType,
            Tables:    tables,
        })
    }
    
    // メインクエリを抽出
    mainQuery = ctePattern.ReplaceAllString(mainQuery, "")
    
    return ctes, mainQuery, nil
}
```

## 5. 出力データ構造

```go
type SQLAnalysisResult struct {
    MethodName string
    Tables     []TableAccess
}

type TableAccess struct {
    TableName  string
    Operations []OperationType
}

// 解析結果をマップとして保持
type AnalysisResults map[string]SQLAnalysisResult // key: メソッド名
```

## 6. エラーハンドリング

### 6.1. パースエラー
```go
type SQLParseError struct {
    Query    string
    Position int
    Message  string
}

func (e SQLParseError) Error() string {
    return fmt.Sprintf("SQL parse error at position %d: %s\nQuery: %s",
        e.Position, e.Message, e.Query)
}
```

### 6.2. 回復可能なエラー
- 未知のSQL構文：警告として記録し、解析を継続
- テーブル名の抽出失敗：空のテーブルリストとして処理

## 7. 最適化

### 7.1. キャッシング
```go
type SQLAnalyzer struct {
    cache map[string]SQLAnalysisResult // SQL文のハッシュをキーとするキャッシュ
    mu    sync.RWMutex
}

func (a *SQLAnalyzer) Analyze(query Query) (SQLAnalysisResult, error) {
    // キャッシュチェック
    hash := hashSQL(query.Text)
    
    a.mu.RLock()
    if result, ok := a.cache[hash]; ok {
        a.mu.RUnlock()
        return result, nil
    }
    a.mu.RUnlock()
    
    // 解析実行
    result, err := a.analyzeInternal(query)
    if err != nil {
        return SQLAnalysisResult{}, err
    }
    
    // キャッシュに保存
    a.mu.Lock()
    a.cache[hash] = result
    a.mu.Unlock()
    
    return result, nil
}
```

### 7.2. 並行処理
```go
func AnalyzeQueries(requests []Query) []SQLAnalysisResult {
    results := make([]SQLAnalysisResult, len(requests))
    var wg sync.WaitGroup
    
    // ワーカープール
    workerCount := runtime.NumCPU()
    jobs := make(chan int, len(requests))
    
    // ワーカー起動
    for w := 0; w < workerCount; w++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for i := range jobs {
                result, _ := analyzer.Analyze(requests[i])
                results[i] = result
            }
        }()
    }
    
    // ジョブ投入
    for i := range requests {
        jobs <- i
    }
    close(jobs)
    
    wg.Wait()
    return results
}
```

## 8. テスト計画

### 8.1. ユニットテスト
- 各SQL操作種別の検出テスト
- テーブル名抽出のテスト
- エッジケース（複雑なJOIN、サブクエリ等）

### 8.2. テストケース例
```go
func TestDetectOperationType(t *testing.T) {
    tests := []struct {
        name     string
        sql      string
        expected OperationType
    }{
        {"Simple SELECT", "SELECT * FROM users", OpSelect},
        {"INSERT with VALUES", "INSERT INTO users VALUES ($1, $2)", OpInsert},
        {"UPDATE with WHERE", "UPDATE users SET name = $1 WHERE id = $2", OpUpdate},
        {"DELETE with JOIN", "DELETE FROM users USING orders WHERE users.id = orders.user_id", OpDelete},
        {"CTE with SELECT", "WITH active_users AS (SELECT * FROM users) SELECT * FROM active_users", OpSelect},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := detectOperationType(tt.sql)
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```