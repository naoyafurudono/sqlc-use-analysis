package output

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/naoyafurudono/sqlc-use-analysis/pkg/types"
)

// Formatter handles output formatting for analysis results
type Formatter struct {
	format types.OutputFormat
	pretty bool
}

// NewFormatter creates a new output formatter
func NewFormatter(format types.OutputFormat, pretty bool) *Formatter {
	return &Formatter{
		format: format,
		pretty: pretty,
	}
}

// Format formats the analysis report according to the specified format
func (f *Formatter) Format(report *types.AnalysisReport, writer io.Writer) error {
	switch f.format {
	case types.FormatJSON:
		return f.formatJSON(report, writer)
	default:
		return fmt.Errorf("unsupported format: %s (only JSON is supported)", f.format)
	}
}

// formatJSON formats the report as JSON
func (f *Formatter) formatJSON(report *types.AnalysisReport, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	
	if f.pretty {
		encoder.SetIndent("", "  ")
	}
	
	// Add metadata
	output := map[string]interface{}{
		"metadata": map[string]interface{}{
			"generated_at": time.Now().Format(time.RFC3339),
			"version":      "1.0.0",
			"tool":         "sqlc-use-analysis",
		},
		"summary":      report.Summary,
		"dependencies": report.Dependencies,
	}
	
	// Add optional sections
	if len(report.Circular) > 0 {
		output["circular_dependencies"] = report.Circular
	}
	
	if len(report.Suggestions) > 0 {
		output["optimization_suggestions"] = report.Suggestions
	}
	
	return encoder.Encode(output)
}
