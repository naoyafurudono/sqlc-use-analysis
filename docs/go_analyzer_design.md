# Go静的解析モジュール 詳細設計

## 1. 概要

Go静的解析モジュールは、Goプロジェクトのソースコードを解析し、以下を実現します：
- 全関数定義の抽出
- 各関数内でのsqlc生成メソッド呼び出しの検出
- 呼び出しチェーンの追跡（間接的な呼び出しも含む）

## 2. 技術選定

### 2.1. 使用するGoパッケージ
- `go/packages`: 型情報を含むパッケージのロード
- `go/ast`: 抽象構文木の操作
- `go/types`: 型情報の取得と解析
- `golang.org/x/tools/go/ssa`: 静的単一代入形式での解析（制御フロー解析用）

## 3. 解析アーキテクチャ

### 3.1. 解析パイプライン

```
Goソースコード
    │
    ▼
┌─────────────────┐
│ ファイル収集器   │ ← 除外パターン適用
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ パッケージローダー│ ← 型情報を含むロード
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ AST解析器       │ ← 関数定義の抽出
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 呼び出し解析器   │ ← メソッド呼び出しの追跡
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 依存グラフ構築器 │ ← 関数間の呼び出し関係
└─────────────────┘
```

## 4. コンポーネント詳細設計

### 4.1. ファイル収集器

```go
type FileCollector struct {
    rootPath string
    excludePatterns []string
}

func (fc *FileCollector) CollectGoFiles() ([]string, error) {
    var files []string
    
    err := filepath.Walk(fc.rootPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        
        // ディレクトリのスキップ判定
        if info.IsDir() {
            if fc.shouldSkipDir(path) {
                return filepath.SkipDir
            }
            return nil
        }
        
        // Goファイルの判定と除外
        if strings.HasSuffix(path, ".go") && !fc.shouldExclude(path) {
            files = append(files, path)
        }
        
        return nil
    })
    
    return files, err
}

func (fc *FileCollector) shouldExclude(path string) bool {
    for _, pattern := range fc.excludePatterns {
        matched, _ := filepath.Match(pattern, path)
        if matched {
            return true
        }
    }
    return false
}
```

### 4.2. パッケージローダー

```go
type PackageLoader struct {
    config *packages.Config
}

func NewPackageLoader() *PackageLoader {
    return &PackageLoader{
        config: &packages.Config{
            Mode: packages.NeedName |
                  packages.NeedFiles |
                  packages.NeedCompiledGoFiles |
                  packages.NeedImports |
                  packages.NeedDeps |
                  packages.NeedTypes |
                  packages.NeedSyntax |
                  packages.NeedTypesInfo,
            Tests: false, // テストファイルは除外
        },
    }
}

func (pl *PackageLoader) Load(patterns ...string) ([]*packages.Package, error) {
    pkgs, err := packages.Load(pl.config, patterns...)
    if err != nil {
        return nil, fmt.Errorf("failed to load packages: %w", err)
    }
    
    // ロードエラーのチェック
    var errs []error
    for _, pkg := range pkgs {
        for _, err := range pkg.Errors {
            errs = append(errs, err)
        }
    }
    
    if len(errs) > 0 {
        return pkgs, fmt.Errorf("package loading errors: %v", errs)
    }
    
    return pkgs, nil
}
```

### 4.3. AST解析器

```go
type ASTAnalyzer struct {
    pkg         *packages.Package
    functions   map[string]*FunctionInfo
    currentFunc *FunctionInfo
}

type FunctionInfo struct {
    PackagePath  string
    Name         string
    Receiver     string // メソッドの場合のレシーバー型
    FilePath     string
    StartLine    int
    EndLine      int
    CalledFuncs  []CallInfo
}

type CallInfo struct {
    PackagePath string
    FuncName    string
    Line        int
}

func (aa *ASTAnalyzer) AnalyzePackage(pkg *packages.Package) map[string]*FunctionInfo {
    aa.pkg = pkg
    aa.functions = make(map[string]*FunctionInfo)
    
    for i, file := range pkg.Syntax {
        aa.analyzeFile(file, pkg.CompiledGoFiles[i])
    }
    
    return aa.functions
}

func (aa *ASTAnalyzer) analyzeFile(file *ast.File, filePath string) {
    ast.Inspect(file, func(n ast.Node) bool {
        switch node := n.(type) {
        case *ast.FuncDecl:
            aa.processFuncDecl(node, filePath)
        case *ast.CallExpr:
            if aa.currentFunc != nil {
                aa.processCallExpr(node)
            }
        }
        return true
    })
}

func (aa *ASTAnalyzer) processFuncDecl(decl *ast.FuncDecl, filePath string) {
    funcName := decl.Name.Name
    var receiver string
    
    // メソッドの場合、レシーバー情報を取得
    if decl.Recv != nil && len(decl.Recv.List) > 0 {
        if t, ok := decl.Recv.List[0].Type.(*ast.StarExpr); ok {
            if ident, ok := t.X.(*ast.Ident); ok {
                receiver = ident.Name
            }
        } else if ident, ok := decl.Recv.List[0].Type.(*ast.Ident); ok {
            receiver = ident.Name
        }
    }
    
    // 完全修飾名の生成
    fullName := aa.pkg.PkgPath
    if receiver != "" {
        fullName += "." + receiver
    }
    fullName += "." + funcName
    
    funcInfo := &FunctionInfo{
        PackagePath: aa.pkg.PkgPath,
        Name:        funcName,
        Receiver:    receiver,
        FilePath:    filePath,
        StartLine:   aa.pkg.Fset.Position(decl.Pos()).Line,
        EndLine:     aa.pkg.Fset.Position(decl.End()).Line,
        CalledFuncs: []CallInfo{},
    }
    
    aa.functions[fullName] = funcInfo
    aa.currentFunc = funcInfo
}
```

### 4.4. 呼び出し解析器

```go
type CallAnalyzer struct {
    typeInfo    *types.Info
    sqlcMethods map[string]bool // sqlc生成メソッドのセット
}

func (ca *CallAnalyzer) processCallExpr(expr *ast.CallExpr, currentFunc *FunctionInfo) {
    // 呼び出し式の型情報を取得
    switch fun := expr.Fun.(type) {
    case *ast.SelectorExpr:
        // メソッド呼び出し (obj.Method())
        ca.processSelectorExpr(fun, expr, currentFunc)
    case *ast.Ident:
        // 関数呼び出し (Function())
        ca.processIdentExpr(fun, expr, currentFunc)
    }
}

func (ca *CallAnalyzer) processSelectorExpr(sel *ast.SelectorExpr, call *ast.CallExpr, currentFunc *FunctionInfo) {
    // セレクタの型情報を取得
    if obj := ca.typeInfo.ObjectOf(sel.Sel); obj != nil {
        if fn, ok := obj.(*types.Func); ok {
            pkg := fn.Pkg()
            if pkg != nil {
                fullName := pkg.Path() + "." + fn.Name()
                
                // sqlc生成メソッドかチェック
                if ca.isSQLCMethod(fullName) {
                    currentFunc.CalledFuncs = append(currentFunc.CalledFuncs, CallInfo{
                        PackagePath: pkg.Path(),
                        FuncName:    fn.Name(),
                        Line:        ca.getLine(call.Pos()),
                    })
                }
            }
        }
    }
}

func (ca *CallAnalyzer) isSQLCMethod(fullName string) bool {
    // sqlc生成パッケージのパスパターンをチェック
    // 例: "project/db/sqlc.GetUser"
    return ca.sqlcMethods[fullName]
}
```

### 4.5. 依存グラフ構築器

```go
type DependencyGraph struct {
    nodes map[string]*GraphNode
    edges map[string][]string
}

type GraphNode struct {
    FunctionName string
    SQLCCalls    []string // 直接呼び出すsqlcメソッド
}

func BuildDependencyGraph(functions map[string]*FunctionInfo) *DependencyGraph {
    graph := &DependencyGraph{
        nodes: make(map[string]*GraphNode),
        edges: make(map[string][]string),
    }
    
    // ノードの作成
    for name, funcInfo := range functions {
        node := &GraphNode{
            FunctionName: name,
            SQLCCalls:    extractSQLCCalls(funcInfo),
        }
        graph.nodes[name] = node
        
        // エッジの作成（関数間の呼び出し関係）
        for _, call := range funcInfo.CalledFuncs {
            calledName := call.PackagePath + "." + call.FuncName
            graph.edges[name] = append(graph.edges[name], calledName)
        }
    }
    
    return graph
}

// 推移的な依存関係の解決
func (dg *DependencyGraph) ResolveTransitiveDependencies() map[string][]string {
    result := make(map[string][]string)
    
    for funcName := range dg.nodes {
        visited := make(map[string]bool)
        sqlcMethods := dg.collectSQLCMethods(funcName, visited)
        if len(sqlcMethods) > 0 {
            result[funcName] = sqlcMethods
        }
    }
    
    return result
}

func (dg *DependencyGraph) collectSQLCMethods(funcName string, visited map[string]bool) []string {
    if visited[funcName] {
        return nil
    }
    visited[funcName] = true
    
    var methods []string
    
    // 直接呼び出すsqlcメソッド
    if node, ok := dg.nodes[funcName]; ok {
        methods = append(methods, node.SQLCCalls...)
    }
    
    // 呼び出す関数から間接的に呼ばれるsqlcメソッド
    if edges, ok := dg.edges[funcName]; ok {
        for _, calledFunc := range edges {
            indirect := dg.collectSQLCMethods(calledFunc, visited)
            methods = append(methods, indirect...)
        }
    }
    
    return methods
}
```

## 5. 最適化戦略

### 5.1. 並行処理

```go
type ConcurrentAnalyzer struct {
    workerCount int
    analyzer    *ASTAnalyzer
}

func (ca *ConcurrentAnalyzer) AnalyzePackages(pkgs []*packages.Package) map[string]*FunctionInfo {
    results := make(map[string]*FunctionInfo)
    resultChan := make(chan map[string]*FunctionInfo, len(pkgs))
    
    // ワーカープール
    sem := make(chan struct{}, ca.workerCount)
    var wg sync.WaitGroup
    
    for _, pkg := range pkgs {
        wg.Add(1)
        sem <- struct{}{} // セマフォ取得
        
        go func(p *packages.Package) {
            defer wg.Done()
            defer func() { <-sem }() // セマフォ解放
            
            analyzer := &ASTAnalyzer{}
            funcs := analyzer.AnalyzePackage(p)
            resultChan <- funcs
        }(pkg)
    }
    
    wg.Wait()
    close(resultChan)
    
    // 結果の集約
    for funcs := range resultChan {
        for name, info := range funcs {
            results[name] = info
        }
    }
    
    return results
}
```

### 5.2. インクリメンタル解析

```go
type IncrementalAnalyzer struct {
    cache      map[string]*CacheEntry
    fileHashes map[string]string
}

type CacheEntry struct {
    Functions  map[string]*FunctionInfo
    LastUpdate time.Time
}

func (ia *IncrementalAnalyzer) ShouldAnalyze(filePath string) bool {
    // ファイルのハッシュを計算
    currentHash, err := ia.calculateFileHash(filePath)
    if err != nil {
        return true // エラー時は再解析
    }
    
    // キャッシュと比較
    if cachedHash, ok := ia.fileHashes[filePath]; ok {
        return currentHash != cachedHash
    }
    
    return true // 初回は解析
}
```

## 6. エラーハンドリング

### 6.1. 解析エラーの分類

```go
type AnalysisError struct {
    Type     ErrorType
    FilePath string
    Line     int
    Message  string
}

type ErrorType int

const (
    ErrorTypeSyntax ErrorType = iota
    ErrorTypeType
    ErrorTypeImport
    ErrorTypeInternal
)

func (ae AnalysisError) Error() string {
    return fmt.Sprintf("%s:%d: %s", ae.FilePath, ae.Line, ae.Message)
}
```

### 6.2. エラー回復戦略

```go
func (aa *ASTAnalyzer) analyzeWithRecovery(pkg *packages.Package) (result map[string]*FunctionInfo, errors []error) {
    defer func() {
        if r := recover(); r != nil {
            errors = append(errors, fmt.Errorf("panic during analysis: %v", r))
        }
    }()
    
    result = aa.AnalyzePackage(pkg)
    
    // パッケージエラーの収集
    for _, err := range pkg.Errors {
        errors = append(errors, AnalysisError{
            Type:     ErrorTypeSyntax,
            FilePath: err.Pos,
            Message:  err.Msg,
        })
    }
    
    return result, errors
}
```

## 7. テスト戦略

### 7.1. テストケース

```go
// テスト用のGoコード例
const testCode = `
package example

import "database/sql"

type Queries struct {
    db *sql.DB
}

func (q *Queries) GetUser(id int64) (*User, error) {
    // sqlc生成メソッドの例
    return nil, nil
}

func BusinessLogic(q *Queries) error {
    user, err := q.GetUser(123) // 直接呼び出し
    if err != nil {
        return err
    }
    return processUser(user)
}

func processUser(u *User) error {
    // 間接的な処理
    return nil
}
`

func TestAnalyzeDirectCall(t *testing.T) {
    // テストコードの解析
    analyzer := NewGoAnalyzer()
    result := analyzer.AnalyzeCode(testCode)
    
    // BusinessLogic関数がGetUserを呼び出していることを確認
    funcInfo := result["example.BusinessLogic"]
    assert.Contains(t, funcInfo.CalledFuncs, "example.Queries.GetUser")
}
```

### 7.2. ベンチマークテスト

```go
func BenchmarkLargeProject(b *testing.B) {
    // 大規模プロジェクトのシミュレーション
    files := generateTestFiles(1000) // 1000ファイル生成
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        analyzer := NewGoAnalyzer()
        analyzer.AnalyzeFiles(files)
    }
}
```