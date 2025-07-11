package io

import (
	"encoding/json"
	"io"
	"os"

	"github.com/naoyafurudono/sqlc-use-analysis/pkg/types"
)

// ResponseWriter writes plugin responses
type ResponseWriter struct {
	writer io.Writer
}

// NewResponseWriter creates a new response writer
func NewResponseWriter() *ResponseWriter {
	return &ResponseWriter{
		writer: os.Stdout,
	}
}

// CodeGeneratorResponse represents the response from a code generator
type CodeGeneratorResponse struct {
	Files []*types.GeneratedFile `json:"files"`
}

// WriteResponse writes the plugin response
func (rw *ResponseWriter) WriteResponse(files []*types.GeneratedFile) error {
	response := &CodeGeneratorResponse{
		Files: files,
	}
	
	encoder := json.NewEncoder(rw.writer)
	return encoder.Encode(response)
}