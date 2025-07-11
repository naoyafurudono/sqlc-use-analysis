package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/naoyafurudono/sqlc-use-analysis/pkg/types"
)

// ConfigLoader loads configuration from various sources
type ConfigLoader struct {
	defaultConfig *types.Config
	envPrefix     string
}

// NewConfigLoader creates a new config loader
func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{
		defaultConfig: DefaultConfig(),
		envPrefix:     "SQLC_ANALYZER_",
	}
}

// CodeGeneratorRequest represents a simplified version of sqlc's request
type CodeGeneratorRequest struct {
	Settings map[string]interface{} `json:"settings"`
	Queries  []interface{}          `json:"queries"`
}

// LoadFromRequest loads configuration from a CodeGeneratorRequest
func (cl *ConfigLoader) LoadFromRequest(request *CodeGeneratorRequest) (*types.Config, error) {
	config := cl.defaultConfig
	
	// プラグインオプションから読み込み
	if request.Settings != nil {
		if err := cl.loadFromPluginOptions(config, request.Settings); err != nil {
			return nil, fmt.Errorf("failed to load plugin options: %w", err)
		}
	}
	
	// 環境変数から読み込み（オーバーライド）
	if err := cl.loadFromEnv(config); err != nil {
		return nil, fmt.Errorf("failed to load from env: %w", err)
	}
	
	// 設定の検証
	if err := cl.validate(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	// 正規化
	cl.normalize(config)
	
	return config, nil
}

func (cl *ConfigLoader) loadFromPluginOptions(config *types.Config, options map[string]interface{}) error {
	// JSONに変換してから構造体にマッピング
	jsonBytes, err := json.Marshal(options)
	if err != nil {
		return err
	}
	
	// 既存の設定にマージ
	return json.Unmarshal(jsonBytes, config)
}

func (cl *ConfigLoader) loadFromEnv(config *types.Config) error {
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

func (cl *ConfigLoader) validate(config *types.Config) error {
	if config.RootPath == "" {
		return fmt.Errorf("root_path cannot be empty")
	}
	
	if config.OutputPath == "" {
		return fmt.Errorf("output_path cannot be empty")
	}
	
	if config.Performance.MaxWorkers < 1 {
		return fmt.Errorf("max_workers must be at least 1")
	}
	
	return nil
}

func (cl *ConfigLoader) normalize(config *types.Config) {
	// パスの正規化は後で実装
	// 今はそのまま
}