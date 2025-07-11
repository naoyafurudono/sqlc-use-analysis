# 設定管理システム 詳細設計

## 1. 概要

設定管理システムは、sqlcプラグインの動作を制御する設定の読み込み、検証、管理を担当します。

## 2. 設定の階層

### 2.1. 設定優先順位

```
1. コマンドライン引数（将来的な拡張）
2. 環境変数
3. sqlc.yamlのプラグイン設定
4. デフォルト値
```

## 3. 設定構造

### 3.1. 基本設定

```go
package config

type Config struct {
    // 基本設定
    RootPath   string   `json:"root_path" yaml:"root_path"`
    OutputPath string   `json:"output_path" yaml:"output_path"`
    Exclude    []string `json:"exclude" yaml:"exclude"`
    
    // 解析設定
    Analysis AnalysisConfig `json:"analysis" yaml:"analysis"`
    
    // 出力設定
    Output OutputConfig `json:"output" yaml:"output"`
    
    // パフォーマンス設定
    Performance PerformanceConfig `json:"performance" yaml:"performance"`
    
    // デバッグ設定
    Debug DebugConfig `json:"debug" yaml:"debug"`
}

type AnalysisConfig struct {
    // Go解析設定
    IncludeTests       bool     `json:"include_tests" yaml:"include_tests"`
    IncludeVendor      bool     `json:"include_vendor" yaml:"include_vendor"`
    FollowSymlinks     bool     `json:"follow_symlinks" yaml:"follow_symlinks"`
    MaxDepth           int      `json:"max_depth" yaml:"max_depth"`
    
    // SQL解析設定
    SQLDialect         string   `json:"sql_dialect" yaml:"sql_dialect"`
    CaseSensitiveTables bool    `json:"case_sensitive_tables" yaml:"case_sensitive_tables"`
    
    // フィルタリング
    IncludePackages    []string `json:"include_packages" yaml:"include_packages"`
    ExcludePackages    []string `json:"exclude_packages" yaml:"exclude_packages"`
}

type OutputConfig struct {
    Format            OutputFormat `json:"format" yaml:"format"`
    IncludeMetadata   bool        `json:"include_metadata" yaml:"include_metadata"`
    IncludeDetails    bool        `json:"include_details" yaml:"include_details"`
    Pretty            bool        `json:"pretty" yaml:"pretty"`
    SplitFiles        bool        `json:"split_files" yaml:"split_files"`
}

type PerformanceConfig struct {
    MaxWorkers        int  `json:"max_workers" yaml:"max_workers"`
    EnableCache       bool `json:"enable_cache" yaml:"enable_cache"`
    MemoryLimit       int  `json:"memory_limit_mb" yaml:"memory_limit_mb"`
    TimeoutSeconds    int  `json:"timeout_seconds" yaml:"timeout_seconds"`
}

type DebugConfig struct {
    Verbose          bool   `json:"verbose" yaml:"verbose"`
    LogFile          string `json:"log_file" yaml:"log_file"`
    ProfileOutput    string `json:"profile_output" yaml:"profile_output"`
    TraceCallPaths   bool   `json:"trace_call_paths" yaml:"trace_call_paths"`
}

type OutputFormat string

const (
    FormatJSON OutputFormat = "json"
    FormatCSV  OutputFormat = "csv"
    FormatHTML OutputFormat = "html"
)
```

### 3.2. デフォルト値

```go
func DefaultConfig() *Config {
    return &Config{
        RootPath:   ".",
        OutputPath: "db_dependencies.json",
        Exclude: []string{
            "**/*_test.go",
            "**/testdata/**",
            "**/vendor/**",
            "**/.git/**",
        },
        Analysis: AnalysisConfig{
            IncludeTests:        false,
            IncludeVendor:       false,
            FollowSymlinks:      false,
            MaxDepth:            10,
            SQLDialect:          "postgresql",
            CaseSensitiveTables: false,
        },
        Output: OutputConfig{
            Format:          FormatJSON,
            IncludeMetadata: true,
            IncludeDetails:  false,
            Pretty:          true,
            SplitFiles:      false,
        },
        Performance: PerformanceConfig{
            MaxWorkers:     runtime.NumCPU(),
            EnableCache:    true,
            MemoryLimit:    1024, // 1GB
            TimeoutSeconds: 300,  // 5 minutes
        },
        Debug: DebugConfig{
            Verbose:        false,
            TraceCallPaths: false,
        },
    }
}
```

## 4. 設定ローダー

### 4.1. 統合ローダー

```go
type ConfigLoader struct {
    defaultConfig *Config
    envPrefix     string
}

func NewConfigLoader() *ConfigLoader {
    return &ConfigLoader{
        defaultConfig: DefaultConfig(),
        envPrefix:     "SQLC_ANALYZER_",
    }
}

func (cl *ConfigLoader) Load(pluginOptions map[string]interface{}) (*Config, error) {
    config := cl.defaultConfig
    
    // 1. プラグインオプションから読み込み
    if err := cl.loadFromPluginOptions(config, pluginOptions); err != nil {
        return nil, fmt.Errorf("failed to load plugin options: %w", err)
    }
    
    // 2. 環境変数から読み込み（オーバーライド）
    if err := cl.loadFromEnv(config); err != nil {
        return nil, fmt.Errorf("failed to load from env: %w", err)
    }
    
    // 3. 設定の検証
    if err := cl.validate(config); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }
    
    // 4. 正規化
    cl.normalize(config)
    
    return config, nil
}
```

### 4.2. プラグインオプションからの読み込み

```go
func (cl *ConfigLoader) loadFromPluginOptions(config *Config, options map[string]interface{}) error {
    // JSONに変換してから構造体にマッピング
    jsonBytes, err := json.Marshal(options)
    if err != nil {
        return err
    }
    
    // 既存の設定にマージ
    return json.Unmarshal(jsonBytes, config)
}
```

### 4.3. 環境変数からの読み込み

```go
func (cl *ConfigLoader) loadFromEnv(config *Config) error {
    // 基本設定
    if v := os.Getenv(cl.envPrefix + "ROOT_PATH"); v != "" {
        config.RootPath = v
    }
    
    if v := os.Getenv(cl.envPrefix + "OUTPUT_PATH"); v != "" {
        config.OutputPath = v
    }
    
    // 除外パターン（カンマ区切り）
    if v := os.Getenv(cl.envPrefix + "EXCLUDE"); v != "" {
        config.Exclude = strings.Split(v, ",")
    }
    
    // パフォーマンス設定
    if v := os.Getenv(cl.envPrefix + "MAX_WORKERS"); v != "" {
        if workers, err := strconv.Atoi(v); err == nil {
            config.Performance.MaxWorkers = workers
        }
    }
    
    // デバッグ設定
    if v := os.Getenv(cl.envPrefix + "VERBOSE"); v != "" {
        config.Debug.Verbose = v == "true" || v == "1"
    }
    
    return nil
}
```

## 5. 設定検証

### 5.1. バリデーター

```go
type ConfigValidator struct {
    errors []ValidationError
}

type ValidationError struct {
    Field   string
    Message string
}

func (cv *ConfigValidator) Validate(config *Config) error {
    cv.errors = nil
    
    // 基本設定の検証
    cv.validateBasic(config)
    
    // 解析設定の検証
    cv.validateAnalysis(&config.Analysis)
    
    // 出力設定の検証
    cv.validateOutput(&config.Output)
    
    // パフォーマンス設定の検証
    cv.validatePerformance(&config.Performance)
    
    if len(cv.errors) > 0 {
        return cv.formatErrors()
    }
    
    return nil
}

func (cv *ConfigValidator) validateBasic(config *Config) {
    // ルートパスの検証
    if config.RootPath == "" {
        cv.addError("root_path", "root path cannot be empty")
    } else if !isValidPath(config.RootPath) {
        cv.addError("root_path", "invalid root path")
    }
    
    // 出力パスの検証
    if config.OutputPath == "" {
        cv.addError("output_path", "output path cannot be empty")
    }
    
    // 除外パターンの検証
    for i, pattern := range config.Exclude {
        if _, err := filepath.Match(pattern, "test"); err != nil {
            cv.addError(fmt.Sprintf("exclude[%d]", i), 
                fmt.Sprintf("invalid glob pattern: %s", pattern))
        }
    }
}

func (cv *ConfigValidator) validateAnalysis(analysis *AnalysisConfig) {
    // 最大深度の検証
    if analysis.MaxDepth < 1 || analysis.MaxDepth > 100 {
        cv.addError("analysis.max_depth", "max depth must be between 1 and 100")
    }
    
    // SQLダイアレクトの検証
    validDialects := []string{"postgresql", "mysql", "sqlite"}
    if !contains(validDialects, analysis.SQLDialect) {
        cv.addError("analysis.sql_dialect", 
            fmt.Sprintf("unsupported SQL dialect: %s", analysis.SQLDialect))
    }
}

func (cv *ConfigValidator) validatePerformance(perf *PerformanceConfig) {
    // ワーカー数の検証
    if perf.MaxWorkers < 1 {
        cv.addError("performance.max_workers", "max workers must be at least 1")
    } else if perf.MaxWorkers > 100 {
        cv.addError("performance.max_workers", "max workers cannot exceed 100")
    }
    
    // メモリ制限の検証
    if perf.MemoryLimit < 0 {
        cv.addError("performance.memory_limit_mb", "memory limit cannot be negative")
    }
    
    // タイムアウトの検証
    if perf.TimeoutSeconds < 0 {
        cv.addError("performance.timeout_seconds", "timeout cannot be negative")
    }
}
```

## 6. 設定の正規化

### 6.1. パスの正規化

```go
func (cl *ConfigLoader) normalize(config *Config) {
    // 絶対パスに変換
    if !filepath.IsAbs(config.RootPath) {
        if abs, err := filepath.Abs(config.RootPath); err == nil {
            config.RootPath = abs
        }
    }
    
    // 出力パスの正規化
    if !filepath.IsAbs(config.OutputPath) {
        config.OutputPath = filepath.Join(config.RootPath, config.OutputPath)
    }
    
    // 除外パターンの正規化
    for i, pattern := range config.Exclude {
        config.Exclude[i] = filepath.Clean(pattern)
    }
    
    // パッケージパスの正規化
    config.Analysis.IncludePackages = normalizePackages(config.Analysis.IncludePackages)
    config.Analysis.ExcludePackages = normalizePackages(config.Analysis.ExcludePackages)
}

func normalizePackages(packages []string) []string {
    normalized := make([]string, len(packages))
    for i, pkg := range packages {
        // 末尾のスラッシュを除去
        normalized[i] = strings.TrimSuffix(pkg, "/")
    }
    return normalized
}
```

## 7. 動的設定

### 7.1. ランタイム設定調整

```go
type RuntimeConfig struct {
    base *Config
    mu   sync.RWMutex
}

func NewRuntimeConfig(base *Config) *RuntimeConfig {
    return &RuntimeConfig{
        base: base,
    }
}

// メモリ使用量に基づいてワーカー数を動的調整
func (rc *RuntimeConfig) AdjustWorkers() {
    rc.mu.Lock()
    defer rc.mu.Unlock()
    
    var memStats runtime.MemStats
    runtime.ReadMemStats(&memStats)
    
    // メモリ使用率が高い場合はワーカー数を減らす
    usedMB := int(memStats.Alloc / 1024 / 1024)
    if usedMB > rc.base.Performance.MemoryLimit*8/10 { // 80%超過
        rc.base.Performance.MaxWorkers = max(1, rc.base.Performance.MaxWorkers/2)
    }
}

// 進捗に基づいてタイムアウトを延長
func (rc *RuntimeConfig) ExtendTimeout(progress float64) {
    rc.mu.Lock()
    defer rc.mu.Unlock()
    
    if progress < 0.5 { // 50%未満の進捗
        estimatedTotal := time.Duration(float64(rc.base.Performance.TimeoutSeconds) / progress)
        rc.base.Performance.TimeoutSeconds = int(estimatedTotal.Seconds() * 1.2) // 20%のマージン
    }
}
```

## 8. 設定のエクスポート

### 8.1. 設定の保存

```go
func (config *Config) Save(path string) error {
    // YAML形式で保存
    data, err := yaml.Marshal(config)
    if err != nil {
        return fmt.Errorf("failed to marshal config: %w", err)
    }
    
    return os.WriteFile(path, data, 0644)
}

// 実行時の設定をログに記録
func (config *Config) LogConfig(logger *log.Logger) {
    logger.Println("=== Configuration ===")
    logger.Printf("Root Path: %s", config.RootPath)
    logger.Printf("Output Path: %s", config.OutputPath)
    logger.Printf("Exclude Patterns: %v", config.Exclude)
    logger.Printf("Max Workers: %d", config.Performance.MaxWorkers)
    logger.Printf("Include Tests: %v", config.Analysis.IncludeTests)
    logger.Println("===================")
}
```

## 9. テスト

### 9.1. 設定ローダーのテスト

```go
func TestConfigLoader(t *testing.T) {
    tests := []struct {
        name    string
        options map[string]interface{}
        env     map[string]string
        want    *Config
        wantErr bool
    }{
        {
            name: "default config",
            options: map[string]interface{}{},
            want: DefaultConfig(),
        },
        {
            name: "custom root path",
            options: map[string]interface{}{
                "root_path": "/custom/path",
            },
            want: &Config{
                RootPath: "/custom/path",
                // ... その他はデフォルト
            },
        },
        {
            name: "env override",
            options: map[string]interface{}{
                "root_path": "/from/options",
            },
            env: map[string]string{
                "SQLC_ANALYZER_ROOT_PATH": "/from/env",
            },
            want: &Config{
                RootPath: "/from/env",
                // ... 環境変数が優先される
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 環境変数の設定
            for k, v := range tt.env {
                os.Setenv(k, v)
                defer os.Unsetenv(k)
            }
            
            loader := NewConfigLoader()
            got, err := loader.Load(tt.options)
            
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.want.RootPath, got.RootPath)
            }
        })
    }
}
```