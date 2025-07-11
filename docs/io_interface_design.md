# 入出力インターフェース 詳細設計

## 1. 概要

入出力インターフェースは、sqlcプラグインとしての標準入出力処理と、解析結果のJSON出力を担当します。

## 2. sqlcプラグインプロトコル

### 2.1. 入力処理

```go
package io

import (
    "encoding/json"
    "io"
    "os"
    
    "github.com/sqlc-dev/sqlc/codegen"
)

type InputReader struct {
    reader io.Reader
}

func NewInputReader() *InputReader {
    return &InputReader{
        reader: os.Stdin,
    }
}

func (ir *InputReader) ReadRequest() (*codegen.CodeGeneratorRequest, error) {
    // 標準入力からJSONを読み込み
    var request codegen.CodeGeneratorRequest
    decoder := json.NewDecoder(ir.reader)
    
    if err := decoder.Decode(&request); err != nil {
        return nil, fmt.Errorf("failed to decode request: %w", err)
    }
    
    // 必須フィールドの検証
    if err := ir.validateRequest(&request); err != nil {
        return nil, fmt.Errorf("invalid request: %w", err)
    }
    
    return &request, nil
}

func (ir *InputReader) validateRequest(req *codegen.CodeGeneratorRequest) error {
    if req.Settings == nil {
        return fmt.Errorf("settings is required")
    }
    
    if len(req.Queries) == 0 {
        return fmt.Errorf("at least one query is required")
    }
    
    return nil
}
```

### 2.2. プラグイン設定の抽出

```go
type PluginConfig struct {
    RootPath   string   `json:"root_path"`
    OutputPath string   `json:"output_path"`
    Exclude    []string `json:"exclude"`
}

func ExtractPluginConfig(request *codegen.CodeGeneratorRequest) (*PluginConfig, error) {
    // sqlc.yamlのプラグイン設定から抽出
    rawConfig := request.Settings.Codegen.Options
    
    configBytes, err := json.Marshal(rawConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal config: %w", err)
    }
    
    var config PluginConfig
    if err := json.Unmarshal(configBytes, &config); err != nil {
        return nil, fmt.Errorf("failed to unmarshal plugin config: %w", err)
    }
    
    // デフォルト値の適用
    if config.RootPath == "" {
        config.RootPath = "."
    }
    
    if config.OutputPath == "" {
        config.OutputPath = "db_dependencies.json"
    }
    
    return &config, nil
}
```

## 3. 出力フォーマット

### 3.1. JSON出力構造

```go
type OutputWriter struct {
    config *PluginConfig
}

type OutputFormat struct {
    Metadata     Metadata                   `json:"metadata"`
    FunctionView map[string][]TableAccess   `json:"function_view"`
    TableView    map[string][]FunctionAccess `json:"table_view"`
}

type Metadata struct {
    GeneratedAt string `json:"generated_at"`
    Version     string `json:"version"`
    TotalFuncs  int    `json:"total_functions"`
    TotalTables int    `json:"total_tables"`
}

func (ow *OutputWriter) WriteResult(result *DependencyResult) error {
    // メタデータの追加
    output := OutputFormat{
        Metadata: Metadata{
            GeneratedAt: time.Now().UTC().Format(time.RFC3339),
            Version:     Version,
            TotalFuncs:  len(result.FunctionView),
            TotalTables: len(result.TableView),
        },
        FunctionView: result.FunctionView,
        TableView:    result.TableView,
    }
    
    // JSON生成（インデント付き）
    jsonBytes, err := json.MarshalIndent(output, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal output: %w", err)
    }
    
    // ファイルへの書き込み
    outputPath := filepath.Join(ow.config.RootPath, ow.config.OutputPath)
    if err := ow.ensureDir(outputPath); err != nil {
        return fmt.Errorf("failed to create output directory: %w", err)
    }
    
    if err := os.WriteFile(outputPath, jsonBytes, 0644); err != nil {
        return fmt.Errorf("failed to write output file: %w", err)
    }
    
    return nil
}

func (ow *OutputWriter) ensureDir(filePath string) error {
    dir := filepath.Dir(filePath)
    return os.MkdirAll(dir, 0755)
}
```

### 3.2. プラグインレスポンス

```go
type ResponseWriter struct {
    writer io.Writer
}

func NewResponseWriter() *ResponseWriter {
    return &ResponseWriter{
        writer: os.Stdout,
    }
}

func (rw *ResponseWriter) WriteResponse(files []*codegen.File) error {
    response := &codegen.CodeGeneratorResponse{
        Files: files,
    }
    
    encoder := json.NewEncoder(rw.writer)
    return encoder.Encode(response)
}

// sqlcプラグインは生成ファイルを返す必要があるため、
// 解析結果を擬似的なファイルとして返す
func CreateDummyFile() *codegen.File {
    return &codegen.File{
        Name:     ".sqlc_dependency_analysis",
        Contents: []byte("// Analysis completed successfully"),
    }
}
```

## 4. 詳細出力フォーマット

### 4.1. 拡張出力モード

```go
type DetailedOutput struct {
    OutputFormat
    Details      Details                 `json:"details,omitempty"`
    Diagnostics  []Diagnostic            `json:"diagnostics,omitempty"`
}

type Details struct {
    CallPaths    map[string][]CallPath   `json:"call_paths,omitempty"`
    UnusedTables []string                `json:"unused_tables,omitempty"`
    Cycles       [][]string              `json:"cycles,omitempty"`
}

type Diagnostic struct {
    Level   string `json:"level"` // "error", "warning", "info"
    Message string `json:"message"`
    Context string `json:"context,omitempty"`
}

func (ow *OutputWriter) WriteDetailedResult(result *DependencyResult, 
    details *AnalysisDetails) error {
    
    output := DetailedOutput{
        OutputFormat: OutputFormat{
            Metadata:     ow.createMetadata(result),
            FunctionView: result.FunctionView,
            TableView:    result.TableView,
        },
        Details:     details.ToOutputFormat(),
        Diagnostics: details.Diagnostics,
    }
    
    // 詳細モードではより読みやすいフォーマットで出力
    jsonBytes, err := json.MarshalIndent(output, "", "    ")
    if err != nil {
        return err
    }
    
    return ow.writeToFile(jsonBytes)
}
```

### 4.2. CSV出力サポート（オプション）

```go
type CSVWriter struct {
    config *PluginConfig
}

func (cw *CSVWriter) WriteFunctionView(result *DependencyResult) error {
    outputPath := strings.TrimSuffix(cw.config.OutputPath, ".json") + "_functions.csv"
    file, err := os.Create(outputPath)
    if err != nil {
        return err
    }
    defer file.Close()
    
    writer := csv.NewWriter(file)
    defer writer.Flush()
    
    // ヘッダー
    writer.Write([]string{"Function", "Table", "Operations"})
    
    // データ
    for funcName, accesses := range result.FunctionView {
        for _, access := range accesses {
            operations := strings.Join(access.Operations, ";")
            writer.Write([]string{funcName, access.Table, operations})
        }
    }
    
    return nil
}
```

## 5. ストリーミング出力

### 5.1. 大規模プロジェクト対応

```go
type StreamingWriter struct {
    writer     io.Writer
    encoder    *json.Encoder
    firstEntry bool
}

func NewStreamingWriter(outputPath string) (*StreamingWriter, error) {
    file, err := os.Create(outputPath)
    if err != nil {
        return nil, err
    }
    
    sw := &StreamingWriter{
        writer:     file,
        encoder:    json.NewEncoder(file),
        firstEntry: true,
    }
    
    // JSON開始
    sw.writer.Write([]byte("{\n"))
    sw.writer.Write([]byte(`  "function_view": {` + "\n"))
    
    return sw, nil
}

func (sw *StreamingWriter) WriteFunction(funcName string, accesses []TableAccess) error {
    if !sw.firstEntry {
        sw.writer.Write([]byte(",\n"))
    }
    sw.firstEntry = false
    
    // 個別のエントリを書き込み
    entry := map[string][]TableAccess{funcName: accesses}
    jsonBytes, err := json.MarshalIndent(entry, "    ", "  ")
    if err != nil {
        return err
    }
    
    // マップの括弧を除去して書き込み
    content := string(jsonBytes)
    content = strings.TrimPrefix(content, "{\n")
    content = strings.TrimSuffix(content, "\n}")
    
    sw.writer.Write([]byte(content))
    return nil
}

func (sw *StreamingWriter) Close() error {
    // JSONを閉じる
    sw.writer.Write([]byte("\n  }\n}"))
    
    if closer, ok := sw.writer.(io.Closer); ok {
        return closer.Close()
    }
    return nil
}
```

## 6. 進捗レポート

### 6.1. 標準エラー出力への進捗表示

```go
type ProgressReporter struct {
    totalSteps   int
    currentStep  int
    startTime    time.Time
    lastReported time.Time
}

func NewProgressReporter(totalSteps int) *ProgressReporter {
    return &ProgressReporter{
        totalSteps:   totalSteps,
        currentStep:  0,
        startTime:    time.Now(),
        lastReported: time.Now(),
    }
}

func (pr *ProgressReporter) Report(message string) {
    pr.currentStep++
    
    // 1秒ごとまたは重要なステップで報告
    if time.Since(pr.lastReported) > time.Second || 
       pr.currentStep == pr.totalSteps {
        
        percentage := float64(pr.currentStep) / float64(pr.totalSteps) * 100
        elapsed := time.Since(pr.startTime)
        
        fmt.Fprintf(os.Stderr, "[%3.0f%%] %s (elapsed: %s)\n", 
            percentage, message, elapsed.Round(time.Second))
        
        pr.lastReported = time.Now()
    }
}
```

## 7. エラー出力

### 7.1. 構造化エラー出力

```go
type ErrorOutput struct {
    Errors []ErrorDetail `json:"errors"`
}

type ErrorDetail struct {
    Type     string                 `json:"type"`
    Message  string                 `json:"message"`
    Location *Location              `json:"location,omitempty"`
    Context  map[string]interface{} `json:"context,omitempty"`
}

type Location struct {
    File string `json:"file"`
    Line int    `json:"line"`
}

func WriteError(err error) {
    var errors []ErrorDetail
    
    // エラーの種類に応じて詳細を構築
    switch e := err.(type) {
    case *ParseError:
        errors = append(errors, ErrorDetail{
            Type:    "parse_error",
            Message: e.Error(),
            Location: &Location{
                File: e.File,
                Line: e.Line,
            },
        })
    default:
        errors = append(errors, ErrorDetail{
            Type:    "general_error",
            Message: err.Error(),
        })
    }
    
    output := ErrorOutput{Errors: errors}
    jsonBytes, _ := json.MarshalIndent(output, "", "  ")
    
    fmt.Fprintln(os.Stderr, string(jsonBytes))
    os.Exit(1)
}
```

## 8. テスト

### 8.1. 出力テスト

```go
func TestJSONOutput(t *testing.T) {
    result := &DependencyResult{
        FunctionView: map[string][]TableAccess{
            "api.Handler": {
                {Table: "users", Operations: []string{"SELECT"}},
            },
        },
        TableView: map[string][]FunctionAccess{
            "users": {
                {Function: "api.Handler", Operations: []string{"SELECT"}},
            },
        },
    }
    
    // 一時ファイルに出力
    tmpDir := t.TempDir()
    config := &PluginConfig{
        OutputPath: filepath.Join(tmpDir, "output.json"),
    }
    
    writer := &OutputWriter{config: config}
    err := writer.WriteResult(result)
    assert.NoError(t, err)
    
    // 出力の検証
    data, err := os.ReadFile(config.OutputPath)
    assert.NoError(t, err)
    
    var output OutputFormat
    err = json.Unmarshal(data, &output)
    assert.NoError(t, err)
    
    assert.Equal(t, 1, output.Metadata.TotalFuncs)
    assert.Equal(t, 1, output.Metadata.TotalTables)
}
```