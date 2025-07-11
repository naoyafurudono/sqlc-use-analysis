package config

import (
	"runtime"

	"github.com/naoyafurudono/sqlc-use-analysis/pkg/types"
)

// DefaultConfig returns the default configuration
func DefaultConfig() *types.Config {
	return &types.Config{
		RootPath:   ".",
		OutputPath: "db_dependencies.json",
		Exclude: []string{
			"**/*_test.go",
			"**/testdata/**",
			"**/vendor/**",
			"**/.git/**",
		},
		Analysis: types.AnalysisConfig{
			IncludeTests:        false,
			IncludeVendor:       false,
			FollowSymlinks:      false,
			MaxDepth:            10,
			SQLDialect:          "postgresql",
			CaseSensitiveTables: false,
		},
		Output: types.OutputConfig{
			Format:          types.FormatJSON,
			IncludeMetadata: true,
			IncludeDetails:  false,
			Pretty:          true,
			SplitFiles:      false,
		},
		Performance: types.PerformanceConfig{
			MaxWorkers:     runtime.NumCPU(),
			EnableCache:    true,
			MemoryLimit:    1024, // 1GB
			TimeoutSeconds: 300,  // 5 minutes
		},
		Debug: types.DebugConfig{
			Verbose:        false,
			TraceCallPaths: false,
		},
	}
}