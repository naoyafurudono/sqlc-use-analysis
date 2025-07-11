package types

// Config represents the plugin configuration
type Config struct {
	// 基本設定
	RootPath   string   `json:"root_path" yaml:"root_path"`
	OutputPath string   `json:"output_path" yaml:"output_path"`
	Exclude    []string `json:"exclude" yaml:"exclude"`
	
	// Go パッケージパス
	GoPackagePaths []string `json:"go_package_paths" yaml:"go_package_paths"`
	
	// 解析設定
	Analysis AnalysisConfig `json:"analysis" yaml:"analysis"`
	
	// 出力設定
	Output OutputConfig `json:"output" yaml:"output"`
	
	// パフォーマンス設定
	Performance PerformanceConfig `json:"performance" yaml:"performance"`
	
	// デバッグ設定
	Debug DebugConfig `json:"debug" yaml:"debug"`
}

// AnalysisConfig contains analysis-specific configuration
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

// OutputConfig contains output-specific configuration
type OutputConfig struct {
	Format            OutputFormat `json:"format" yaml:"format"`
	IncludeMetadata   bool        `json:"include_metadata" yaml:"include_metadata"`
	IncludeDetails    bool        `json:"include_details" yaml:"include_details"`
	Pretty            bool        `json:"pretty" yaml:"pretty"`
	SplitFiles        bool        `json:"split_files" yaml:"split_files"`
}

// PerformanceConfig contains performance-related configuration
type PerformanceConfig struct {
	MaxWorkers        int  `json:"max_workers" yaml:"max_workers"`
	EnableCache       bool `json:"enable_cache" yaml:"enable_cache"`
	MemoryLimit       int  `json:"memory_limit_mb" yaml:"memory_limit_mb"`
	TimeoutSeconds    int  `json:"timeout_seconds" yaml:"timeout_seconds"`
}

// DebugConfig contains debug-related configuration
type DebugConfig struct {
	Verbose          bool   `json:"verbose" yaml:"verbose"`
	LogFile          string `json:"log_file" yaml:"log_file"`
	ProfileOutput    string `json:"profile_output" yaml:"profile_output"`
	TraceCallPaths   bool   `json:"trace_call_paths" yaml:"trace_call_paths"`
}

// OutputFormat represents the output format
type OutputFormat string

const (
	FormatJSON OutputFormat = "json"
	FormatCSV  OutputFormat = "csv"
	FormatHTML OutputFormat = "html"
)