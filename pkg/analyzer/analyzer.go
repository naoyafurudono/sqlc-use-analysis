// Package analyzer provides a deep, simple interface for dependency analysis
// Following "A Philosophy of Software Design" principles
package analyzer

import (
	"context"
	"fmt"

	"github.com/naoyafurudono/sqlc-use-analysis/internal/analyzer/dependency"
	"github.com/naoyafurudono/sqlc-use-analysis/internal/errors"
	"github.com/naoyafurudono/sqlc-use-analysis/internal/output"
	"github.com/naoyafurudono/sqlc-use-analysis/pkg/types"
)

// Query represents a SQL query to be analyzed
type Query struct {
	Name string `json:"name"`
	SQL  string `json:"sql"`
}

// AnalysisRequest contains all inputs needed for analysis
type AnalysisRequest struct {
	SQLQueries   []Query  `json:"sql_queries"`
	GoPackages   []string `json:"go_packages"`
	OutputFormat string   `json:"output_format,omitempty"` // "json", "csv", "html"
	PrettyPrint  bool     `json:"pretty_print,omitempty"`
}

// Result represents the complete analysis result
type Result struct {
	Functions    map[string]FunctionInfo  `json:"functions"`
	Tables       map[string]TableInfo     `json:"tables"`
	Dependencies []Dependency             `json:"dependencies"`
	Summary      Summary                  `json:"summary"`
	Suggestions  []OptimizationTip        `json:"suggestions,omitempty"`
}

// FunctionInfo represents information about a Go function
type FunctionInfo struct {
	Name        string            `json:"name"`
	Package     string            `json:"package"`
	File        string            `json:"file"`
	StartLine   int               `json:"start_line"`
	EndLine     int               `json:"end_line"`
	TableAccess map[string]Access `json:"table_access"`
}

// TableInfo represents information about a database table
type TableInfo struct {
	Name          string            `json:"name"`
	AccessedBy    []string          `json:"accessed_by"`
	OperationCount map[string]int   `json:"operation_count"`
}

// Dependency represents a dependency between a function and a table
type Dependency struct {
	Function  string `json:"function"`
	Table     string `json:"table"`
	Operation string `json:"operation"`
	Method    string `json:"method"`
	Line      int    `json:"line"`
}

// Access represents how a function accesses a table
type Access struct {
	Operations []string `json:"operations"`
	Methods    []string `json:"methods"`
	Count      int      `json:"count"`
}

// Summary provides high-level statistics
type Summary struct {
	FunctionCount   int            `json:"function_count"`
	TableCount      int            `json:"table_count"`
	DependencyCount int            `json:"dependency_count"`
	OperationCounts map[string]int `json:"operation_counts"`
}

// OptimizationTip provides actionable optimization suggestions
type OptimizationTip struct {
	Type        string `json:"type"`
	Function    string `json:"function,omitempty"`
	Table       string `json:"table,omitempty"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

// Analyzer provides a deep module for dependency analysis
// It hides all complexity behind a simple interface
type Analyzer struct {
	engine *dependency.Engine
	errors *errors.ErrorCollector
}

// New creates a new analyzer with sensible defaults
// This is the only way to create an analyzer, ensuring proper initialization
func New() *Analyzer {
	errorCollector := errors.NewErrorCollector(100, false)
	return &Analyzer{
		engine: dependency.NewEngine(errorCollector),
		errors: errorCollector,
	}
}

// Analyze performs complete dependency analysis
// This is the main interface - all complexity is hidden inside
func (a *Analyzer) Analyze(ctx context.Context, request AnalysisRequest) (*Result, error) {
	// Input validation
	if err := a.validateRequest(request); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Convert external types to internal types
	queries := a.convertQueries(request.SQLQueries)
	
	// Perform the analysis using the internal engine
	// All engine complexity is hidden from the caller
	result, err := a.engine.AnalyzeDependencies(queries, request.GoPackages)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	// Convert internal result to external format
	// This transformation hides internal complexity
	analysisResult := a.convertResult(result)
	
	return analysisResult, nil
}

// AnalyzeAndFormat performs analysis and returns formatted output
// This combines analysis and formatting in a single call for convenience
func (a *Analyzer) AnalyzeAndFormat(ctx context.Context, request AnalysisRequest) ([]byte, error) {
	result, err := a.Analyze(ctx, request)
	if err != nil {
		return nil, err
	}

	// Determine output format
	format := request.OutputFormat
	if format == "" {
		format = "json"
	}

	// Convert to internal format types
	var outputFormat types.OutputFormat
	switch format {
	case "json":
		outputFormat = types.FormatJSON
	default:
		return nil, fmt.Errorf("unsupported output format: %s (only JSON is supported)", format)
	}

	// Format the result
	// Note: This is a simplified implementation for demonstration
	// In practice, you'd use the formatter to generate actual output
	_ = output.NewFormatter(outputFormat, request.PrettyPrint)
	_ = a.convertToReport(result)
	
	// For now, return a simple JSON representation
	// TODO: Implement proper formatting
	return []byte(`{"status": "analysis_complete"}`), nil
}

// GetErrors returns any errors that occurred during analysis
// This provides access to detailed error information if needed
func (a *Analyzer) GetErrors() []AnalysisError {
	internalErrors := a.errors.GetAllErrors()
	externalErrors := make([]AnalysisError, len(internalErrors))
	
	for i, err := range internalErrors {
		externalErrors[i] = AnalysisError{
			ID       : err.ID,
			Category : string(err.Category),
			Severity : err.Severity.String(),
			Message  : err.Message,
			Details  : err.Details,
		}
	}
	
	return externalErrors
}

// AnalysisError represents an error that occurred during analysis
type AnalysisError struct {
	ID       string                 `json:"id"`
	Category string                 `json:"category"`
	Severity string                 `json:"severity"`
	Message  string                 `json:"message"`
	Details  map[string]interface{} `json:"details,omitempty"`
}

// Helper methods (private, hiding complexity)

func (a *Analyzer) validateRequest(request AnalysisRequest) error {
	if len(request.SQLQueries) == 0 {
		return fmt.Errorf("no SQL queries provided")
	}
	
	if len(request.GoPackages) == 0 {
		return fmt.Errorf("no Go packages provided")
	}
	
	for i, query := range request.SQLQueries {
		if query.Name == "" {
			return fmt.Errorf("query %d has empty name", i)
		}
		if query.SQL == "" {
			return fmt.Errorf("query '%s' has empty SQL", query.Name)
		}
	}
	
	return nil
}

func (a *Analyzer) convertQueries(queries []Query) []types.QueryInfo {
	converted := make([]types.QueryInfo, len(queries))
	for i, q := range queries {
		converted[i] = types.QueryInfo{
			Name: q.Name,
			SQL:  q.SQL,
		}
	}
	return converted
}

func (a *Analyzer) convertResult(internalResult types.AnalysisResult) *Result {
	result := &Result{
		Functions:    make(map[string]FunctionInfo),
		Tables:       make(map[string]TableInfo),
		Dependencies: []Dependency{},
		Summary: Summary{
			OperationCounts: make(map[string]int),
		},
	}
	
	// Convert function view
	for funcName, funcEntry := range internalResult.FunctionView {
		funcInfo := FunctionInfo{
			Name:        funcEntry.FunctionName,
			Package:     funcEntry.PackageName,
			File:        funcEntry.FileName,
			StartLine:   funcEntry.StartLine,
			EndLine:     funcEntry.EndLine,
			TableAccess: make(map[string]Access),
		}
		
		// Convert table access information
		for tableName, tableAccess := range funcEntry.TableAccess {
			access := Access{
				Operations: []string{},
				Methods:    []string{},
				Count:      0,
			}
			
			for operation, calls := range tableAccess.Operations {
				access.Operations = append(access.Operations, operation)
				access.Count += len(calls)
				
				for _, call := range calls {
					access.Methods = append(access.Methods, call.MethodName)
					
					// Create dependency entry
					result.Dependencies = append(result.Dependencies, Dependency{
						Function:  funcName,
						Table:     tableName,
						Operation: operation,
						Method:    call.MethodName,
						Line:      call.Line,
					})
				}
			}
			
			funcInfo.TableAccess[tableName] = access
		}
		
		result.Functions[funcName] = funcInfo
	}
	
	// Convert table view
	for tableName, tableEntry := range internalResult.TableView {
		accessedBy := make([]string, 0, len(tableEntry.AccessedBy))
		for funcName := range tableEntry.AccessedBy {
			accessedBy = append(accessedBy, funcName)
		}
		
		result.Tables[tableName] = TableInfo{
			Name:           tableName,
			AccessedBy:     accessedBy,
			OperationCount: tableEntry.OperationSummary,
		}
	}
	
	// Calculate summary
	result.Summary.FunctionCount = len(result.Functions)
	result.Summary.TableCount = len(result.Tables)
	result.Summary.DependencyCount = len(result.Dependencies)
	
	// Count operations
	for _, dep := range result.Dependencies {
		result.Summary.OperationCounts[dep.Operation]++
	}
	
	return result
}

func (a *Analyzer) convertToReport(result *Result) *types.AnalysisReport {
	// Convert external result back to internal report format
	// This is needed for the formatter
	report := &types.AnalysisReport{
		Summary: types.AnalysisSummary{
			FunctionCount:   result.Summary.FunctionCount,
			TableCount:      result.Summary.TableCount,
			OperationCounts: result.Summary.OperationCounts,
			PackageCounts:   make(map[string]int),
		},
		Dependencies: types.AnalysisResult{
			FunctionView: make(map[string]types.FunctionViewEntry),
			TableView:    make(map[string]types.TableViewEntry),
		},
		Suggestions: []types.OptimizationSuggestion{},
	}
	
	// This would need full conversion logic, but shows the pattern
	return report
}