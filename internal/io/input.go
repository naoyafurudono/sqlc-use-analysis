package io

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	
	"github.com/naoyafurudono/sqlc-use-analysis/internal/config"
)

// InputReader reads input from various sources
type InputReader struct {
	reader io.Reader
}

// NewInputReader creates a new input reader
func NewInputReader() *InputReader {
	return &InputReader{
		reader: os.Stdin,
	}
}

// ReadRequest reads a CodeGeneratorRequest from the input
func (ir *InputReader) ReadRequest() (*config.CodeGeneratorRequest, error) {
	var request config.CodeGeneratorRequest
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

func (ir *InputReader) validateRequest(req *config.CodeGeneratorRequest) error {
	// 基本的な検証
	if req.Settings == nil {
		// 設定が空の場合はデフォルト値を使用
		req.Settings = make(map[string]interface{})
	}
	
	// クエリが空の場合は警告（エラーではない）
	if len(req.Queries) == 0 {
		// ログ出力などで警告を出す（後で実装）
	}
	
	return nil
}