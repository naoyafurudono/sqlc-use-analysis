package gostatic

import (
	"fmt"
	"strings"

	"github.com/naoyafurudono/sqlc-use-analysis/internal/errors"
	"github.com/naoyafurudono/sqlc-use-analysis/pkg/types"
)

// DependencyMapper maps Go functions to SQL methods and database tables
type DependencyMapper struct {
	errorCollector *errors.ErrorCollector
}

// NewDependencyMapper creates a new dependency mapper
func NewDependencyMapper(errorCollector *errors.ErrorCollector) *DependencyMapper {
	return &DependencyMapper{
		errorCollector: errorCollector,
	}
}

// MapDependencies maps Go functions to SQL methods and creates dependency relationships
func (m *DependencyMapper) MapDependencies(
	goFunctions map[string]types.GoFunctionInfo,
	sqlMethods map[string]types.SQLMethodInfo,
) (types.AnalysisResult, error) {
	
	result := types.AnalysisResult{
		FunctionView: make(map[string]types.FunctionViewEntry),
		TableView:    make(map[string]types.TableViewEntry),
	}

	// Create function view entries
	for funcName, funcInfo := range goFunctions {
		entry := types.FunctionViewEntry{
			FunctionName: funcInfo.FunctionName,
			PackageName:  funcInfo.PackageName,
			FileName:     funcInfo.FileName,
			StartLine:    funcInfo.StartLine,
			EndLine:      funcInfo.EndLine,
			TableAccess:  make(map[string]types.TableAccessInfo),
		}

		// Map SQL calls to table access
		for _, sqlCall := range funcInfo.SQLCalls {
			if sqlMethodInfo, exists := sqlMethods[sqlCall.MethodName]; exists {
				// Add table access for each table in the SQL method
				for _, tableOp := range sqlMethodInfo.Tables {
					m.addTableAccess(&entry, tableOp, sqlCall)
				}
			} else {
				// Log warning for unmapped SQL method
				mapErr := errors.NewError(errors.CategoryMapping, errors.SeverityWarning,
					fmt.Sprintf("SQL method '%s' not found in SQL analysis", sqlCall.MethodName))
				mapErr.Details["function"] = funcInfo.FunctionName
				mapErr.Details["method"] = sqlCall.MethodName
				mapErr.Details["line"] = fmt.Sprintf("%d", sqlCall.Line)

				if collectErr := m.errorCollector.Add(mapErr); collectErr != nil {
					return result, collectErr
				}
			}
		}

		result.FunctionView[funcName] = entry
	}

	// Create table view entries
	result.TableView = m.createTableView(result.FunctionView)

	return result, nil
}

// addTableAccess adds table access information to a function view entry
func (m *DependencyMapper) addTableAccess(
	entry *types.FunctionViewEntry,
	tableOp types.TableOperation,
	sqlCall types.SQLCall,
) {
	tableName := tableOp.TableName
	
	// Get existing table access or create new one
	access, exists := entry.TableAccess[tableName]
	if !exists {
		access = types.TableAccessInfo{
			TableName:  tableName,
			Operations: make(map[string][]types.OperationCall),
		}
	}

	// Add operation calls for each operation type
	for _, operation := range tableOp.Operations {
		opCall := types.OperationCall{
			MethodName: sqlCall.MethodName,
			Line:       sqlCall.Line,
			Column:     sqlCall.Column,
		}

		access.Operations[operation] = append(access.Operations[operation], opCall)
	}

	entry.TableAccess[tableName] = access
}

// createTableView creates table view entries from function view
func (m *DependencyMapper) createTableView(
	functionView map[string]types.FunctionViewEntry,
) map[string]types.TableViewEntry {
	
	tableView := make(map[string]types.TableViewEntry)

	for _, funcEntry := range functionView {
		for tableName, tableAccess := range funcEntry.TableAccess {
			// Get existing table view entry or create new one
			entry, exists := tableView[tableName]
			if !exists {
				entry = types.TableViewEntry{
					TableName:     tableName,
					AccessedBy:    make(map[string]types.FunctionAccess),
					OperationSummary: make(map[string]int),
				}
			}

			// Add function access
			var operations []string
			for operation := range tableAccess.Operations {
				operations = append(operations, operation)
			}
			
			funcAccess := types.FunctionAccess{
				Function:   funcEntry.FunctionName,
				Operations: operations,
			}

			// Update operation summary
			for operation, calls := range tableAccess.Operations {
				entry.OperationSummary[operation] += len(calls)
			}

			entry.AccessedBy[funcEntry.FunctionName] = funcAccess
			tableView[tableName] = entry
		}
	}

	return tableView
}

// ValidateDependencies validates the dependency mapping results
func (m *DependencyMapper) ValidateDependencies(result types.AnalysisResult) error {
	var validationErrors []error

	// Validate function view
	for funcName, funcEntry := range result.FunctionView {
		if funcEntry.FunctionName == "" {
			validationErrors = append(validationErrors, 
				fmt.Errorf("function '%s' has empty function name", funcName))
		}

		if funcEntry.PackageName == "" {
			validationErrors = append(validationErrors, 
				fmt.Errorf("function '%s' has empty package name", funcName))
		}

		// Validate table access
		for tableName, tableAccess := range funcEntry.TableAccess {
			if tableAccess.TableName != tableName {
				validationErrors = append(validationErrors, 
					fmt.Errorf("function '%s' has inconsistent table name: key='%s', value='%s'", 
						funcName, tableName, tableAccess.TableName))
			}

			if len(tableAccess.Operations) == 0 {
				validationErrors = append(validationErrors, 
					fmt.Errorf("function '%s' has no operations for table '%s'", 
						funcName, tableName))
			}
		}
	}

	// Validate table view
	for tableName, tableEntry := range result.TableView {
		if tableEntry.TableName != tableName {
			validationErrors = append(validationErrors, 
				fmt.Errorf("table '%s' has inconsistent table name: key='%s', value='%s'", 
					tableName, tableName, tableEntry.TableName))
		}

		if len(tableEntry.AccessedBy) == 0 {
			validationErrors = append(validationErrors, 
				fmt.Errorf("table '%s' has no accessing functions", tableName))
		}

		// Validate operation summary
		totalOperations := 0
		for _, count := range tableEntry.OperationSummary {
			totalOperations += count
		}

		if totalOperations == 0 {
			validationErrors = append(validationErrors, 
				fmt.Errorf("table '%s' has no operations in summary", tableName))
		}
	}

	// Report validation errors
	if len(validationErrors) > 0 {
		for _, err := range validationErrors {
			valErr := errors.NewError(errors.CategoryValidation, errors.SeverityError, err.Error())
			if collectErr := m.errorCollector.Add(valErr); collectErr != nil {
				return collectErr
			}
		}
		return fmt.Errorf("validation failed with %d errors", len(validationErrors))
	}

	return nil
}

// GenerateSummary generates a summary of the dependency analysis
func (m *DependencyMapper) GenerateSummary(result types.AnalysisResult) types.AnalysisSummary {
	summary := types.AnalysisSummary{
		FunctionCount: len(result.FunctionView),
		TableCount:    len(result.TableView),
		OperationCounts: make(map[string]int),
		PackageCounts:   make(map[string]int),
	}

	// Count operations and packages
	for _, funcEntry := range result.FunctionView {
		// Count package
		summary.PackageCounts[funcEntry.PackageName]++

		// Count operations
		for _, tableAccess := range funcEntry.TableAccess {
			for operation, calls := range tableAccess.Operations {
				summary.OperationCounts[operation] += len(calls)
			}
		}
	}

	return summary
}

// FindCircularDependencies finds circular dependencies in the analysis result
func (m *DependencyMapper) FindCircularDependencies(result types.AnalysisResult) []types.CircularDependency {
	var circular []types.CircularDependency

	// Build dependency graph
	graph := make(map[string][]string)
	
	for funcName, funcEntry := range result.FunctionView {
		for tableName := range funcEntry.TableAccess {
			// In this context, we consider functions that access the same table
			// as potentially having circular dependencies
			if tableEntry, exists := result.TableView[tableName]; exists {
				for otherFuncName := range tableEntry.AccessedBy {
					if otherFuncName != funcName {
						graph[funcName] = append(graph[funcName], otherFuncName)
					}
				}
			}
		}
	}

	// Simple cycle detection (for demonstration)
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	
	var dfs func(string, []string) bool
	dfs = func(node string, path []string) bool {
		visited[node] = true
		recStack[node] = true
		
		for _, neighbor := range graph[node] {
			if !visited[neighbor] {
				if dfs(neighbor, append(path, node)) {
					return true
				}
			} else if recStack[neighbor] {
				// Found cycle
				cycle := append(path, node, neighbor)
				circular = append(circular, types.CircularDependency{
					Functions: cycle,
					Type:      "table_access",
				})
				return true
			}
		}
		
		recStack[node] = false
		return false
	}

	for funcName := range result.FunctionView {
		if !visited[funcName] {
			dfs(funcName, []string{})
		}
	}

	return circular
}

// OptimizeDependencies suggests optimizations for the dependency structure
func (m *DependencyMapper) OptimizeDependencies(result types.AnalysisResult) []types.OptimizationSuggestion {
	var suggestions []types.OptimizationSuggestion

	// Find functions that access many tables
	for funcName, funcEntry := range result.FunctionView {
		if len(funcEntry.TableAccess) > 5 {
			suggestions = append(suggestions, types.OptimizationSuggestion{
				Type:        "high_table_access",
				Function:    funcName,
				Description: fmt.Sprintf("Function accesses %d tables, consider splitting", len(funcEntry.TableAccess)),
				Severity:    "medium",
			})
		}
	}

	// Find tables accessed by many functions
	for tableName, tableEntry := range result.TableView {
		if len(tableEntry.AccessedBy) > 10 {
			suggestions = append(suggestions, types.OptimizationSuggestion{
				Type:        "high_function_access",
				Table:       tableName,
				Description: fmt.Sprintf("Table accessed by %d functions, consider access patterns", len(tableEntry.AccessedBy)),
				Severity:    "low",
			})
		}
	}

	// Find functions with mixed operations on same table
	for funcName, funcEntry := range result.FunctionView {
		for tableName, tableAccess := range funcEntry.TableAccess {
			operations := make([]string, 0, len(tableAccess.Operations))
			for op := range tableAccess.Operations {
				operations = append(operations, op)
			}
			
			if len(operations) > 2 {
				suggestions = append(suggestions, types.OptimizationSuggestion{
					Type:        "mixed_operations",
					Function:    funcName,
					Table:       tableName,
					Description: fmt.Sprintf("Function performs %s operations on table, consider separation", strings.Join(operations, ", ")),
					Severity:    "low",
				})
			}
		}
	}

	return suggestions
}