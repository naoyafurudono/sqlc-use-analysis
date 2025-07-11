# エラーハンドリング方針

## 1. 基本方針

### 1.1. エラー処理の原則

1. **早期検出・早期報告**: エラーは可能な限り早期に検出し、適切なコンテキストと共に報告する
2. **回復可能性の考慮**: 部分的な失敗でも可能な限り処理を継続し、有用な結果を提供する
3. **詳細な診断情報**: デバッグに必要な情報（ファイルパス、行番号、関連データ）を含める
4. **ユーザーフレンドリー**: エラーメッセージは明確で、解決方法を示唆する

### 1.2. エラーの分類

```go
type ErrorSeverity int

const (
    SeverityFatal   ErrorSeverity = iota // 処理続行不可
    SeverityError                        // エラーだが処理は継続可能
    SeverityWarning                      // 警告レベル
    SeverityInfo                         // 情報レベル
)

type ErrorCategory string

const (
    CategoryConfig    ErrorCategory = "CONFIG"     // 設定関連
    CategoryParse     ErrorCategory = "PARSE"      // パース関連
    CategoryAnalysis  ErrorCategory = "ANALYSIS"   // 解析関連
    CategoryIO        ErrorCategory = "IO"         // 入出力関連
    CategoryInternal  ErrorCategory = "INTERNAL"   // 内部エラー
)
```

## 2. エラー構造

### 2.1. 基本エラー型

```go
package errors

import (
    "fmt"
    "runtime"
    "time"
)

type AnalysisError struct {
    ID         string                 // 一意のエラーID
    Category   ErrorCategory          // エラーカテゴリ
    Severity   ErrorSeverity          // 深刻度
    Message    string                 // エラーメッセージ
    Details    map[string]interface{} // 詳細情報
    Location   *ErrorLocation         // エラー発生場所
    Timestamp  time.Time              // 発生時刻
    StackTrace string                 // スタックトレース（デバッグモード時）
    Wrapped    error                  // ラップされた元エラー
}

type ErrorLocation struct {
    File     string // ファイルパス
    Line     int    // 行番号
    Column   int    // 列番号（該当する場合）
    Function string // 関数名
}

func (e *AnalysisError) Error() string {
    if e.Location != nil {
        return fmt.Sprintf("[%s] %s at %s:%d - %s", 
            e.Category, e.ID, e.Location.File, e.Location.Line, e.Message)
    }
    return fmt.Sprintf("[%s] %s - %s", e.Category, e.ID, e.Message)
}

func (e *AnalysisError) Unwrap() error {
    return e.Wrapped
}
```

### 2.2. エラー生成ヘルパー

```go
func NewError(category ErrorCategory, severity ErrorSeverity, message string) *AnalysisError {
    pc, file, line, _ := runtime.Caller(1)
    fn := runtime.FuncForPC(pc)
    
    return &AnalysisError{
        ID:        generateErrorID(),
        Category:  category,
        Severity:  severity,
        Message:   message,
        Timestamp: time.Now(),
        Location: &ErrorLocation{
            File:     file,
            Line:     line,
            Function: fn.Name(),
        },
        Details: make(map[string]interface{}),
    }
}

func Wrap(err error, message string) *AnalysisError {
    if err == nil {
        return nil
    }
    
    // 既存のAnalysisErrorの場合は情報を保持
    if ae, ok := err.(*AnalysisError); ok {
        ae.Message = fmt.Sprintf("%s: %s", message, ae.Message)
        return ae
    }
    
    // 新規エラーとしてラップ
    newErr := NewError(CategoryInternal, SeverityError, message)
    newErr.Wrapped = err
    return newErr
}
```

## 3. エラーコレクター

### 3.1. エラー収集システム

```go
type ErrorCollector struct {
    errors     []*AnalysisError
    warnings   []*AnalysisError
    mu         sync.Mutex
    maxErrors  int
    stopOnFatal bool
}

func NewErrorCollector(maxErrors int, stopOnFatal bool) *ErrorCollector {
    return &ErrorCollector{
        errors:      make([]*AnalysisError, 0),
        warnings:    make([]*AnalysisError, 0),
        maxErrors:   maxErrors,
        stopOnFatal: stopOnFatal,
    }
}

func (ec *ErrorCollector) Add(err *AnalysisError) error {
    ec.mu.Lock()
    defer ec.mu.Unlock()
    
    switch err.Severity {
    case SeverityFatal:
        ec.errors = append(ec.errors, err)
        if ec.stopOnFatal {
            return err // 即座に処理を停止
        }
    case SeverityError:
        ec.errors = append(ec.errors, err)
        if len(ec.errors) >= ec.maxErrors {
            return fmt.Errorf("too many errors: %d", len(ec.errors))
        }
    case SeverityWarning:
        ec.warnings = append(ec.warnings, err)
    }
    
    return nil
}

func (ec *ErrorCollector) HasErrors() bool {
    ec.mu.Lock()
    defer ec.mu.Unlock()
    return len(ec.errors) > 0
}

func (ec *ErrorCollector) GetReport() *ErrorReport {
    ec.mu.Lock()
    defer ec.mu.Unlock()
    
    return &ErrorReport{
        Errors:   ec.errors,
        Warnings: ec.warnings,
        Summary:  ec.generateSummary(),
    }
}
```

## 4. 具体的なエラー処理

### 4.1. 設定エラー

```go
type ConfigError struct {
    *AnalysisError
    Field string
    Value interface{}
}

func NewConfigError(field string, value interface{}, message string) *ConfigError {
    err := NewError(CategoryConfig, SeverityFatal, message)
    err.Details["field"] = field
    err.Details["value"] = value
    
    return &ConfigError{
        AnalysisError: err,
        Field:         field,
        Value:         value,
    }
}

// 使用例
func validateConfig(config *Config) error {
    if config.RootPath == "" {
        return NewConfigError("root_path", "", 
            "root_path is required but not provided")
    }
    
    if !isValidPath(config.RootPath) {
        return NewConfigError("root_path", config.RootPath,
            fmt.Sprintf("invalid path: %s - please ensure the path exists and is accessible", 
                config.RootPath))
    }
    
    return nil
}
```

### 4.2. パースエラー

```go
type ParseError struct {
    *AnalysisError
    SourceCode string
    Position   int
}

func NewParseError(file string, line int, source string, message string) *ParseError {
    err := NewError(CategoryParse, SeverityError, message)
    err.Location = &ErrorLocation{
        File: file,
        Line: line,
    }
    err.Details["source"] = source
    
    return &ParseError{
        AnalysisError: err,
        SourceCode:    source,
    }
}

// エラーコンテキストの表示
func (pe *ParseError) GetContext(contextLines int) string {
    lines := strings.Split(pe.SourceCode, "\n")
    start := max(0, pe.Location.Line-contextLines-1)
    end := min(len(lines), pe.Location.Line+contextLines)
    
    var context strings.Builder
    for i := start; i < end; i++ {
        prefix := "  "
        if i == pe.Location.Line-1 {
            prefix = "> "
        }
        context.WriteString(fmt.Sprintf("%s%4d | %s\n", prefix, i+1, lines[i]))
    }
    
    return context.String()
}
```

### 4.3. 解析エラー

```go
type AnalysisContextError struct {
    *AnalysisError
    Function   string
    Package    string
    CallChain  []string
}

func NewAnalysisError(pkg, function string, message string) *AnalysisContextError {
    err := NewError(CategoryAnalysis, SeverityError, message)
    err.Details["package"] = pkg
    err.Details["function"] = function
    
    return &AnalysisContextError{
        AnalysisError: err,
        Function:      function,
        Package:       pkg,
    }
}

// 循環参照エラー
func NewCyclicDependencyError(cycle []string) *AnalysisContextError {
    err := NewAnalysisError("", "", 
        fmt.Sprintf("cyclic dependency detected: %s", strings.Join(cycle, " -> ")))
    err.Severity = SeverityWarning
    err.Details["cycle"] = cycle
    err.CallChain = cycle
    
    return err
}
```

## 5. エラーリカバリー

### 5.1. パニックリカバリー

```go
func RecoverWithError(collector *ErrorCollector) {
    if r := recover(); r != nil {
        stack := make([]byte, 4096)
        n := runtime.Stack(stack, false)
        
        err := NewError(CategoryInternal, SeverityFatal, 
            fmt.Sprintf("panic recovered: %v", r))
        err.StackTrace = string(stack[:n])
        
        collector.Add(err)
    }
}

// 使用例
func (analyzer *Analyzer) AnalyzePackage(pkg *packages.Package) {
    defer RecoverWithError(analyzer.errorCollector)
    
    // 解析処理...
}
```

### 5.2. 部分的失敗の処理

```go
type PartialResult struct {
    Result      interface{}
    Errors      []*AnalysisError
    IsComplete  bool
}

func ProcessWithPartialFailure(items []string, processor func(string) error) *PartialResult {
    result := &PartialResult{
        Result:     make([]interface{}, 0),
        Errors:     make([]*AnalysisError, 0),
        IsComplete: true,
    }
    
    for _, item := range items {
        if err := processor(item); err != nil {
            if ae, ok := err.(*AnalysisError); ok {
                result.Errors = append(result.Errors, ae)
                if ae.Severity == SeverityFatal {
                    result.IsComplete = false
                    break
                }
            }
        }
    }
    
    return result
}
```

## 6. エラーレポート

### 6.1. エラーレポート生成

```go
type ErrorReport struct {
    Errors   []*AnalysisError
    Warnings []*AnalysisError
    Summary  ErrorSummary
}

type ErrorSummary struct {
    TotalErrors   int
    TotalWarnings int
    ByCategory    map[ErrorCategory]int
    BySeverity    map[ErrorSeverity]int
}

func (er *ErrorReport) Format(format string) string {
    switch format {
    case "json":
        return er.formatJSON()
    case "text":
        return er.formatText()
    case "markdown":
        return er.formatMarkdown()
    default:
        return er.formatText()
    }
}

func (er *ErrorReport) formatText() string {
    var buf strings.Builder
    
    buf.WriteString("=== Error Report ===\n\n")
    
    if len(er.Errors) > 0 {
        buf.WriteString("ERRORS:\n")
        for i, err := range er.Errors {
            buf.WriteString(fmt.Sprintf("%d. %s\n", i+1, err.Error()))
            if err.Location != nil {
                buf.WriteString(fmt.Sprintf("   Location: %s:%d\n", 
                    err.Location.File, err.Location.Line))
            }
            buf.WriteString("\n")
        }
    }
    
    if len(er.Warnings) > 0 {
        buf.WriteString("\nWARNINGS:\n")
        for i, warn := range er.Warnings {
            buf.WriteString(fmt.Sprintf("%d. %s\n", i+1, warn.Message))
        }
    }
    
    buf.WriteString(fmt.Sprintf("\nSummary: %d errors, %d warnings\n", 
        er.Summary.TotalErrors, er.Summary.TotalWarnings))
    
    return buf.String()
}
```

### 6.2. エラーの集約

```go
type ErrorAggregator struct {
    groups map[string][]*AnalysisError
}

func (ea *ErrorAggregator) Add(err *AnalysisError) {
    key := ea.generateKey(err)
    ea.groups[key] = append(ea.groups[key], err)
}

func (ea *ErrorAggregator) generateKey(err *AnalysisError) string {
    // 類似エラーをグループ化するためのキー生成
    return fmt.Sprintf("%s:%s:%s", err.Category, err.ID, err.Message)
}

func (ea *ErrorAggregator) GetAggregatedReport() []AggregatedError {
    var result []AggregatedError
    
    for key, errors := range ea.groups {
        result = append(result, AggregatedError{
            Key:        key,
            Count:      len(errors),
            FirstError: errors[0],
            Locations:  ea.extractLocations(errors),
        })
    }
    
    return result
}
```

## 7. ロギング統合

### 7.1. 構造化ログ

```go
type ErrorLogger struct {
    logger *slog.Logger
}

func (el *ErrorLogger) LogError(err *AnalysisError) {
    attrs := []slog.Attr{
        slog.String("error_id", err.ID),
        slog.String("category", string(err.Category)),
        slog.String("severity", err.Severity.String()),
        slog.Time("timestamp", err.Timestamp),
    }
    
    if err.Location != nil {
        attrs = append(attrs, 
            slog.String("file", err.Location.File),
            slog.Int("line", err.Location.Line))
    }
    
    el.logger.LogAttrs(context.Background(), 
        slog.LevelError, err.Message, attrs...)
}
```

## 8. ユーザー向けエラーメッセージ

### 8.1. エラーメッセージのローカライズ

```go
var errorMessages = map[string]string{
    "CONFIG_MISSING_ROOT": "Configuration error: 'root_path' is required. Please specify the project root directory in your sqlc.yaml file.",
    "PARSE_INVALID_SQL": "SQL parsing error: The query contains invalid SQL syntax. Please check your query definition.",
    "ANALYSIS_CYCLIC_DEP": "Analysis warning: Circular dependency detected in function calls. This may indicate a design issue.",
}

func GetUserFriendlyMessage(err *AnalysisError) string {
    if msg, ok := errorMessages[err.ID]; ok {
        return msg
    }
    return err.Message
}
```

## 9. テスト

### 9.1. エラーハンドリングのテスト

```go
func TestErrorCollector(t *testing.T) {
    collector := NewErrorCollector(10, false)
    
    // エラーの追加
    err1 := NewError(CategoryParse, SeverityError, "test error 1")
    err2 := NewError(CategoryAnalysis, SeverityWarning, "test warning")
    
    collector.Add(err1)
    collector.Add(err2)
    
    report := collector.GetReport()
    
    assert.Equal(t, 1, len(report.Errors))
    assert.Equal(t, 1, len(report.Warnings))
    assert.Equal(t, 1, report.Summary.TotalErrors)
}
```