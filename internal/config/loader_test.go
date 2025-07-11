package config

import (
	"os"
	"testing"
	
	"github.com/naoyafurudono/sqlc-use-analysis/pkg/types"
)

func TestConfigLoader_LoadFromRequest(t *testing.T) {
	tests := []struct {
		name    string
		request *CodeGeneratorRequest
		env     map[string]string
		want    func(*testing.T, *types.Config)
		wantErr bool
	}{
		{
			name: "default config",
			request: &CodeGeneratorRequest{
				Settings: make(map[string]interface{}),
				Queries:  []interface{}{},
			},
			want: func(t *testing.T, cfg *types.Config) {
				if cfg.RootPath != "." {
					t.Errorf("Expected RootPath to be '.', got '%s'", cfg.RootPath)
				}
				if cfg.OutputPath != "db_dependencies.json" {
					t.Errorf("Expected OutputPath to be 'db_dependencies.json', got '%s'", cfg.OutputPath)
				}
			},
		},
		{
			name: "custom root path",
			request: &CodeGeneratorRequest{
				Settings: map[string]interface{}{
					"root_path": "/custom/path",
				},
				Queries: []interface{}{},
			},
			want: func(t *testing.T, cfg *types.Config) {
				if cfg.RootPath != "/custom/path" {
					t.Errorf("Expected RootPath to be '/custom/path', got '%s'", cfg.RootPath)
				}
			},
		},
		{
			name: "env override",
			request: &CodeGeneratorRequest{
				Settings: map[string]interface{}{
					"root_path": "/from/options",
				},
				Queries: []interface{}{},
			},
			env: map[string]string{
				"SQLC_ANALYZER_ROOT_PATH": "/from/env",
			},
			want: func(t *testing.T, cfg *types.Config) {
				if cfg.RootPath != "/from/env" {
					t.Errorf("Expected RootPath to be '/from/env', got '%s'", cfg.RootPath)
				}
			},
		},
		{
			name: "invalid config - empty root path",
			request: &CodeGeneratorRequest{
				Settings: map[string]interface{}{
					"root_path": "",
				},
				Queries: []interface{}{},
			},
			wantErr: true,
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
			got, err := loader.LoadFromRequest(tt.request)
			
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("LoadFromRequest() error = %v", err)
				return
			}
			
			if tt.want != nil {
				tt.want(t, got)
			}
		})
	}
}

func TestConfigLoader_loadFromEnv(t *testing.T) {
	loader := NewConfigLoader()
	config := DefaultConfig()
	
	// 環境変数の設定
	os.Setenv("SQLC_ANALYZER_ROOT_PATH", "/test/path")
	os.Setenv("SQLC_ANALYZER_OUTPUT_PATH", "test_output.json")
	os.Setenv("SQLC_ANALYZER_VERBOSE", "true")
	
	defer func() {
		os.Unsetenv("SQLC_ANALYZER_ROOT_PATH")
		os.Unsetenv("SQLC_ANALYZER_OUTPUT_PATH")
		os.Unsetenv("SQLC_ANALYZER_VERBOSE")
	}()
	
	err := loader.loadFromEnv(config)
	if err != nil {
		t.Errorf("loadFromEnv() error = %v", err)
	}
	
	if config.RootPath != "/test/path" {
		t.Errorf("Expected RootPath to be '/test/path', got '%s'", config.RootPath)
	}
	
	if config.OutputPath != "test_output.json" {
		t.Errorf("Expected OutputPath to be 'test_output.json', got '%s'", config.OutputPath)
	}
	
	if !config.Debug.Verbose {
		t.Error("Expected Verbose to be true")
	}
}