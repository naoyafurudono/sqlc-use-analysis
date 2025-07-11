# 依存関係マッピングエンジン 詳細設計

## 1. 概要

依存関係マッピングエンジンは、SQLクエリ解析結果とGo静的解析結果を統合し、以下を実現します：
- Go関数とデータベーステーブルの関係を特定
- 直接的・間接的な依存関係の解決
- function_viewとtable_viewの双方向マッピング生成

## 2. マッピング戦略

### 2.1. 依存関係の種類

```go
type DependencyType int

const (
    DirectDependency   DependencyType = iota // 直接sqlcメソッドを呼び出す
    IndirectDependency                       // 他の関数を経由して呼び出す
)

type DependencyPath struct {
    From         string   // 起点の関数
    To           string   // sqlcメソッド
    Intermediate []string // 経由する関数のリスト
    Type         DependencyType
}
```

### 2.2. マッピングプロセス

```
┌──────────────────┐     ┌──────────────────┐
│ SQL解析結果       │     │ Go解析結果        │
│ - メソッド名      │     │ - 関数定義        │
│ - テーブル名      │     │ - 呼び出し関係    │
│ - 操作種別        │     │                  │
└────────┬─────────┘     └────────┬─────────┘
         │                         │
         └───────────┬─────────────┘
                     │
                     ▼
         ┌───────────────────────┐
         │ 依存関係グラフ構築      │
         └───────────┬───────────┘
                     │
                     ▼
         ┌───────────────────────┐
         │ 推移的依存関係の解決    │
         └───────────┬───────────┘
                     │
                     ▼
         ┌───────────────────────┐
         │ 双方向ビューの生成      │
         │ - function_view       │
         │ - table_view          │
         └───────────────────────┘
```

## 3. コア実装

### 3.1. マッピングエンジン

```go
type DependencyMapper struct {
    sqlAnalysis  map[string]SQLMethodInfo    // sqlcメソッド → テーブル操作
    goAnalysis   map[string]*GoFunctionInfo  // Go関数 → 呼び出し情報
    callGraph    *CallGraph                  // 関数呼び出しグラフ
}

type SQLMethodInfo struct {
    MethodName string
    Tables     []TableOperation
}

type TableOperation struct {
    TableName  string
    Operations []string // SELECT, INSERT, UPDATE, DELETE
}

type GoFunctionInfo struct {
    FullName      string
    DirectCalls   []string // 直接呼び出すメソッド
    AllCalls      []string // 推移的に呼び出すメソッド（計算後）
}
```

### 3.2. 呼び出しグラフの構築

```go
type CallGraph struct {
    nodes map[string]*CallNode
    edges map[string][]*CallEdge
}

type CallNode struct {
    FunctionName string
    IsSQLCMethod bool
    TableOps     []TableOperation // sqlcメソッドの場合のみ
}

type CallEdge struct {
    From string
    To   string
    Line int
}

func (dm *DependencyMapper) BuildCallGraph() *CallGraph {
    graph := &CallGraph{
        nodes: make(map[string]*CallNode),
        edges: make(map[string][]*CallEdge),
    }
    
    // sqlcメソッドのノード作成
    for methodName, sqlInfo := range dm.sqlAnalysis {
        graph.nodes[methodName] = &CallNode{
            FunctionName: methodName,
            IsSQLCMethod: true,
            TableOps:     sqlInfo.Tables,
        }
    }
    
    // Go関数のノード作成とエッジ構築
    for funcName, funcInfo := range dm.goAnalysis {
        graph.nodes[funcName] = &CallNode{
            FunctionName: funcName,
            IsSQLCMethod: false,
        }
        
        // 直接呼び出しのエッジ作成
        for _, calledMethod := range funcInfo.DirectCalls {
            edge := &CallEdge{
                From: funcName,
                To:   calledMethod,
            }
            graph.edges[funcName] = append(graph.edges[funcName], edge)
        }
    }
    
    return graph
}
```

### 3.3. 推移的依存関係の解決

```go
func (dm *DependencyMapper) ResolveTransitiveDependencies() {
    // 各関数について、到達可能なsqlcメソッドを探索
    for funcName, funcInfo := range dm.goAnalysis {
        reachableMethods := dm.findReachableSQLCMethods(funcName)
        funcInfo.AllCalls = reachableMethods
    }
}

func (dm *DependencyMapper) findReachableSQLCMethods(startFunc string) []string {
    visited := make(map[string]bool)
    sqlcMethods := make(map[string]bool)
    
    // 深さ優先探索
    var dfs func(current string)
    dfs = func(current string) {
        if visited[current] {
            return
        }
        visited[current] = true
        
        // 現在のノードがsqlcメソッドかチェック
        if node, ok := dm.callGraph.nodes[current]; ok && node.IsSQLCMethod {
            sqlcMethods[current] = true
        }
        
        // 呼び出し先を探索
        if edges, ok := dm.callGraph.edges[current]; ok {
            for _, edge := range edges {
                dfs(edge.To)
            }
        }
    }
    
    dfs(startFunc)
    
    // 結果をスライスに変換
    result := make([]string, 0, len(sqlcMethods))
    for method := range sqlcMethods {
        result = append(result, method)
    }
    
    return result
}
```

### 3.4. パス追跡（デバッグ用）

```go
type CallPath struct {
    Steps []CallStep
}

type CallStep struct {
    Function string
    Line     int
}

func (dm *DependencyMapper) FindPaths(from, to string) []CallPath {
    var paths []CallPath
    var currentPath []CallStep
    visited := make(map[string]bool)
    
    dm.findPathsDFS(from, to, visited, currentPath, &paths)
    
    return paths
}

func (dm *DependencyMapper) findPathsDFS(current, target string, visited map[string]bool, 
    currentPath []CallStep, paths *[]CallPath) {
    
    if current == target {
        // パスを発見
        pathCopy := make([]CallStep, len(currentPath))
        copy(pathCopy, currentPath)
        *paths = append(*paths, CallPath{Steps: pathCopy})
        return
    }
    
    if visited[current] {
        return
    }
    
    visited[current] = true
    defer func() { visited[current] = false }() // バックトラック
    
    if edges, ok := dm.callGraph.edges[current]; ok {
        for _, edge := range edges {
            step := CallStep{
                Function: edge.To,
                Line:     edge.Line,
            }
            currentPath = append(currentPath, step)
            dm.findPathsDFS(edge.To, target, visited, currentPath, paths)
            currentPath = currentPath[:len(currentPath)-1] // バックトラック
        }
    }
}
```

### 3.5. ビューの生成

```go
type DependencyResult struct {
    FunctionView map[string][]TableAccess `json:"function_view"`
    TableView    map[string][]FunctionAccess `json:"table_view"`
}

type TableAccess struct {
    Table      string   `json:"table"`
    Operations []string `json:"operations"`
}

type FunctionAccess struct {
    Function   string   `json:"function"`
    Operations []string `json:"operations"`
}

func (dm *DependencyMapper) GenerateViews() *DependencyResult {
    result := &DependencyResult{
        FunctionView: make(map[string][]TableAccess),
        TableView:    make(map[string][]FunctionAccess),
    }
    
    // Function Viewの構築
    for funcName, funcInfo := range dm.goAnalysis {
        tableMap := make(map[string]map[string]bool) // table -> operations
        
        // 関数が呼び出す全sqlcメソッドを走査
        for _, sqlcMethod := range funcInfo.AllCalls {
            if sqlInfo, ok := dm.sqlAnalysis[sqlcMethod]; ok {
                for _, tableOp := range sqlInfo.Tables {
                    if tableMap[tableOp.TableName] == nil {
                        tableMap[tableOp.TableName] = make(map[string]bool)
                    }
                    for _, op := range tableOp.Operations {
                        tableMap[tableOp.TableName][op] = true
                    }
                }
            }
        }
        
        // マップをスライスに変換
        if len(tableMap) > 0 {
            var tableAccesses []TableAccess
            for table, ops := range tableMap {
                operations := make([]string, 0, len(ops))
                for op := range ops {
                    operations = append(operations, op)
                }
                sort.Strings(operations) // 安定した出力のためソート
                
                tableAccesses = append(tableAccesses, TableAccess{
                    Table:      table,
                    Operations: operations,
                })
            }
            sort.Slice(tableAccesses, func(i, j int) bool {
                return tableAccesses[i].Table < tableAccesses[j].Table
            })
            result.FunctionView[funcName] = tableAccesses
        }
    }
    
    // Table Viewの構築（FunctionViewの逆引き）
    for funcName, tableAccesses := range result.FunctionView {
        for _, ta := range tableAccesses {
            funcAccess := FunctionAccess{
                Function:   funcName,
                Operations: ta.Operations,
            }
            result.TableView[ta.Table] = append(result.TableView[ta.Table], funcAccess)
        }
    }
    
    // Table Viewのソート
    for table, funcs := range result.TableView {
        sort.Slice(funcs, func(i, j int) bool {
            return funcs[i].Function < funcs[j].Function
        })
        result.TableView[table] = funcs
    }
    
    return result
}
```

## 4. 最適化

### 4.1. グラフ探索の最適化

```go
type OptimizedMapper struct {
    *DependencyMapper
    reachabilityCache map[string]map[string]bool // from -> to -> reachable
    cacheMutex        sync.RWMutex
}

func (om *OptimizedMapper) IsReachable(from, to string) bool {
    // キャッシュチェック
    om.cacheMutex.RLock()
    if toMap, ok := om.reachabilityCache[from]; ok {
        if reachable, ok := toMap[to]; ok {
            om.cacheMutex.RUnlock()
            return reachable
        }
    }
    om.cacheMutex.RUnlock()
    
    // 計算
    reachable := om.computeReachability(from, to)
    
    // キャッシュ更新
    om.cacheMutex.Lock()
    if om.reachabilityCache[from] == nil {
        om.reachabilityCache[from] = make(map[string]bool)
    }
    om.reachabilityCache[from][to] = reachable
    om.cacheMutex.Unlock()
    
    return reachable
}
```

### 4.2. 並行マッピング

```go
func (dm *DependencyMapper) GenerateViewsConcurrent() *DependencyResult {
    result := &DependencyResult{
        FunctionView: make(map[string][]TableAccess),
        TableView:    make(map[string][]FunctionAccess),
    }
    
    // Function Viewを並行生成
    funcNames := make([]string, 0, len(dm.goAnalysis))
    for name := range dm.goAnalysis {
        funcNames = append(funcNames, name)
    }
    
    type funcResult struct {
        funcName string
        accesses []TableAccess
    }
    
    resultChan := make(chan funcResult, len(funcNames))
    sem := make(chan struct{}, runtime.NumCPU())
    
    var wg sync.WaitGroup
    for _, funcName := range funcNames {
        wg.Add(1)
        sem <- struct{}{}
        
        go func(fn string) {
            defer wg.Done()
            defer func() { <-sem }()
            
            accesses := dm.computeTableAccesses(fn)
            resultChan <- funcResult{
                funcName: fn,
                accesses: accesses,
            }
        }(funcName)
    }
    
    wg.Wait()
    close(resultChan)
    
    // 結果の集約
    var mu sync.Mutex
    for res := range resultChan {
        if len(res.accesses) > 0 {
            mu.Lock()
            result.FunctionView[res.funcName] = res.accesses
            mu.Unlock()
        }
    }
    
    // Table Viewの構築（シングルスレッド）
    dm.buildTableView(result)
    
    return result
}
```

## 5. 循環参照の検出

```go
type CycleDetector struct {
    graph *CallGraph
}

func (cd *CycleDetector) DetectCycles() [][]string {
    var cycles [][]string
    visited := make(map[string]int) // 0: 未訪問, 1: 訪問中, 2: 訪問済
    var path []string
    
    for node := range cd.graph.nodes {
        if visited[node] == 0 {
            cd.dfsDetectCycle(node, visited, &path, &cycles)
        }
    }
    
    return cycles
}

func (cd *CycleDetector) dfsDetectCycle(node string, visited map[string]int, 
    path *[]string, cycles *[][]string) {
    
    visited[node] = 1 // 訪問中
    *path = append(*path, node)
    
    if edges, ok := cd.graph.edges[node]; ok {
        for _, edge := range edges {
            if visited[edge.To] == 1 {
                // 循環を検出
                cycleStart := 0
                for i, n := range *path {
                    if n == edge.To {
                        cycleStart = i
                        break
                    }
                }
                cycle := make([]string, len(*path)-cycleStart)
                copy(cycle, (*path)[cycleStart:])
                *cycles = append(*cycles, cycle)
            } else if visited[edge.To] == 0 {
                cd.dfsDetectCycle(edge.To, visited, path, cycles)
            }
        }
    }
    
    *path = (*path)[:len(*path)-1]
    visited[node] = 2 // 訪問済
}
```

## 6. エラーハンドリング

```go
type MappingError struct {
    Type    MappingErrorType
    Context string
    Details map[string]interface{}
}

type MappingErrorType int

const (
    ErrorUnresolvedMethod MappingErrorType = iota
    ErrorCyclicDependency
    ErrorInconsistentData
)

func (dm *DependencyMapper) Validate() []MappingError {
    var errors []MappingError
    
    // sqlcメソッドの存在確認
    for _, funcInfo := range dm.goAnalysis {
        for _, call := range funcInfo.DirectCalls {
            if _, ok := dm.sqlAnalysis[call]; !ok {
                errors = append(errors, MappingError{
                    Type:    ErrorUnresolvedMethod,
                    Context: funcInfo.FullName,
                    Details: map[string]interface{}{
                        "method": call,
                    },
                })
            }
        }
    }
    
    // 循環参照の検出
    detector := &CycleDetector{graph: dm.callGraph}
    cycles := detector.DetectCycles()
    for _, cycle := range cycles {
        errors = append(errors, MappingError{
            Type:    ErrorCyclicDependency,
            Context: "call graph",
            Details: map[string]interface{}{
                "cycle": cycle,
            },
        })
    }
    
    return errors
}
```

## 7. テスト設計

```go
func TestComplexDependencyMapping(t *testing.T) {
    // テスト用のデータ構造
    sqlAnalysis := map[string]SQLMethodInfo{
        "db.GetUser": {
            MethodName: "GetUser",
            Tables: []TableOperation{
                {TableName: "users", Operations: []string{"SELECT"}},
            },
        },
        "db.UpdateUser": {
            MethodName: "UpdateUser",
            Tables: []TableOperation{
                {TableName: "users", Operations: []string{"UPDATE"}},
            },
        },
    }
    
    goAnalysis := map[string]*GoFunctionInfo{
        "api.GetUserHandler": {
            FullName:    "api.GetUserHandler",
            DirectCalls: []string{"service.GetUserByID"},
        },
        "service.GetUserByID": {
            FullName:    "service.GetUserByID",
            DirectCalls: []string{"db.GetUser"},
        },
    }
    
    mapper := &DependencyMapper{
        sqlAnalysis: sqlAnalysis,
        goAnalysis:  goAnalysis,
    }
    
    // マッピング実行
    mapper.BuildCallGraph()
    mapper.ResolveTransitiveDependencies()
    result := mapper.GenerateViews()
    
    // 検証
    assert.Contains(t, result.FunctionView, "api.GetUserHandler")
    assert.Equal(t, "users", result.FunctionView["api.GetUserHandler"][0].Table)
    assert.Contains(t, result.FunctionView["api.GetUserHandler"][0].Operations, "SELECT")
}
```