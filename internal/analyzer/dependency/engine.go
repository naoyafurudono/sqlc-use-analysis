package dependency

import (
	"fmt"
	"path/filepath"
	"strings"

	gostatic "github.com/naoyafurudono/sqlc-use-analysis/internal/analyzer/go"
	"github.com/naoyafurudono/sqlc-use-analysis/internal/analyzer/sql"
	"github.com/naoyafurudono/sqlc-use-analysis/internal/errors"
	"github.com/naoyafurudono/sqlc-use-analysis/pkg/types"
)

// Engine orchestrates the complete dependency analysis
type Engine struct {
	sqlAnalyzer    *sql.Analyzer
	goAnalyzer     *gostatic.Analyzer
	mapper         *gostatic.DependencyMapper
	errorCollector *errors.ErrorCollector
}

// NewEngine creates a new dependency analysis engine
func NewEngine(errorCollector *errors.ErrorCollector) *Engine {
	return &Engine{
		sqlAnalyzer:    sql.NewAnalyzer("mysql", false, errorCollector),
		errorCollector: errorCollector,
	}
}

// AnalyzeDependencies performs complete dependency analysis
func (e *Engine) AnalyzeDependencies(
	sqlQueries []types.QueryInfo,
	goPackagePaths []string,
) (types.AnalysisResult, error) {
	
	// Step 1: Analyze SQL queries to extract method and table information
	sqlMethods, err := e.analyzeSQLQueries(sqlQueries)
	if err != nil {
		return types.AnalysisResult{}, fmt.Errorf("SQL analysis failed: %w", err)
	}

	// Step 2: Analyze Go code to extract function and method call information
	goFunctions, err := e.analyzeGoCode(goPackagePaths)
	if err != nil {
		return types.AnalysisResult{}, fmt.Errorf("Go analysis failed: %w", err)
	}

	// Step 3: Map dependencies between Go functions and SQL methods
	e.mapper = gostatic.NewDependencyMapper(e.errorCollector)
	result, err := e.mapper.MapDependencies(goFunctions, sqlMethods)
	if err != nil {
		return types.AnalysisResult{}, fmt.Errorf("dependency mapping failed: %w", err)
	}

	// Step 4: Validate the mapping results
	if err := e.mapper.ValidateDependencies(result); err != nil {
		return types.AnalysisResult{}, fmt.Errorf("dependency validation failed: %w", err)
	}

	return result, nil
}

// analyzeSQLQueries analyzes SQL queries and extracts method information
func (e *Engine) analyzeSQLQueries(queries []types.QueryInfo) (map[string]types.SQLMethodInfo, error) {
	sqlMethods := make(map[string]types.SQLMethodInfo)
	reporter := errors.NewErrorReporter(e.errorCollector)

	for _, query := range queries {
		// Create SQL Query object
		sqlQuery := sql.Query{
			Text:     query.SQL,
			Name:     query.Name,
			Cmd:      ":exec", // Default command
			Filename: "",
		}

		// Analyze the SQL query
		analysisResult, err := e.sqlAnalyzer.AnalyzeQuery(sqlQuery)
		if err != nil {
			// Log error but continue processing using the new error helper
			queryReporter := reporter.WithQueryContext(query.Name, query.SQL)
			if collectErr := queryReporter.Error(errors.CategoryAnalysis, 
				fmt.Sprintf("failed to analyze SQL query: %v", err)); collectErr != nil {
				return nil, collectErr
			}
			continue
		}

		// The analysisResult is already a SQLMethodInfo, so use it directly
		sqlMethods[analysisResult.MethodName] = analysisResult
	}

	return sqlMethods, nil
}

// analyzeGoCode analyzes Go source code and extracts function information
func (e *Engine) analyzeGoCode(packagePaths []string) (map[string]types.GoFunctionInfo, error) {
	if len(packagePaths) == 0 {
		return make(map[string]types.GoFunctionInfo), nil
	}

	// Initialize Go analyzer
	e.goAnalyzer = gostatic.NewAnalyzer(".", e.errorCollector)

	// Load packages
	if err := e.goAnalyzer.LoadPackages(packagePaths...); err != nil {
		return nil, fmt.Errorf("failed to load Go packages: %w", err)
	}

	// Analyze packages
	functions, err := e.goAnalyzer.AnalyzePackages()
	if err != nil {
		return nil, fmt.Errorf("failed to analyze Go packages: %w", err)
	}

	return functions, nil
}

// GenerateReport generates a comprehensive analysis report
func (e *Engine) GenerateReport(result types.AnalysisResult) types.AnalysisReport {
	report := types.AnalysisReport{
		Summary:      e.mapper.GenerateSummary(result),
		Dependencies: result,
		Circular:     e.mapper.FindCircularDependencies(result),
		Suggestions:  e.mapper.OptimizeDependencies(result),
	}

	return report
}

// ValidateInput validates the input parameters
func (e *Engine) ValidateInput(queries []types.QueryInfo, packagePaths []string) error {
	if len(queries) == 0 {
		return fmt.Errorf("no SQL queries provided")
	}

	if len(packagePaths) == 0 {
		return fmt.Errorf("no Go package paths provided")
	}

	// Validate SQL queries
	for i, query := range queries {
		if query.Name == "" {
			return fmt.Errorf("query at index %d has empty name", i)
		}
		if query.SQL == "" {
			return fmt.Errorf("query '%s' has empty SQL", query.Name)
		}
	}

	// Validate package paths
	for i, path := range packagePaths {
		if path == "" {
			return fmt.Errorf("package path at index %d is empty", i)
		}
		if !isValidPackagePath(path) {
			return fmt.Errorf("invalid package path: %s", path)
		}
	}

	return nil
}

// isValidPackagePath checks if a package path is valid
func isValidPackagePath(path string) bool {
	// Basic validation - could be enhanced
	if path == "" {
		return false
	}
	
	if strings.Contains(path, "../") {
		return false
	}
	
	if filepath.IsAbs(path) {
		return true
	}
	
	// Relative paths and Go module paths
	return true
}

// GetStats returns analysis statistics
func (e *Engine) GetStats() EngineStats {
	return EngineStats{
		ErrorCount:       e.errorCollector.Count(),
		HasErrors:        e.errorCollector.HasErrors(),
		HasWarnings:      e.errorCollector.HasWarnings(),
		ErrorsByCategory: e.getErrorsByCategory(),
	}
}

// getErrorsByCategory groups errors by category
func (e *Engine) getErrorsByCategory() map[string]int {
	categoryCounts := make(map[string]int)
	
	for _, err := range e.errorCollector.GetAllErrors() {
		categoryCounts[string(err.Category)]++
	}
	
	return categoryCounts
}

// EngineStats represents analysis engine statistics
type EngineStats struct {
	ErrorCount       int            `json:"error_count"`
	HasErrors        bool           `json:"has_errors"`
	HasWarnings      bool           `json:"has_warnings"`
	ErrorsByCategory map[string]int `json:"errors_by_category"`
}

// Reset clears the engine state for reuse
func (e *Engine) Reset() {
	e.errorCollector.Clear()
	e.sqlAnalyzer = sql.NewAnalyzer("mysql", false, e.errorCollector)
	e.goAnalyzer = nil
	e.mapper = nil
}

// SetMaxErrors sets the maximum number of errors to collect
func (e *Engine) SetMaxErrors(maxErrors int) {
	e.errorCollector = errors.NewErrorCollector(maxErrors, e.errorCollector.IsDebugMode())
}

// EnableDebugMode enables debug mode for detailed error information
func (e *Engine) EnableDebugMode() {
	e.errorCollector = errors.NewErrorCollector(e.errorCollector.GetMaxErrors(), true)
}