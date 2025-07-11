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
	case types.FormatCSV:
		return f.formatCSV(report, writer)
	case types.FormatHTML:
		return f.formatHTML(report, writer)
	default:
		return fmt.Errorf("unsupported format: %s", f.format)
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

// formatCSV formats the report as CSV
func (f *Formatter) formatCSV(report *types.AnalysisReport, writer io.Writer) error {
	// Function view CSV
	fmt.Fprintf(writer, "# Function View\n")
	fmt.Fprintf(writer, "Function,Package,File,Tables,Operations\n")
	
	for _, funcEntry := range report.Dependencies.FunctionView {
		var tables []string
		var operations []string
		
		for tableName, tableAccess := range funcEntry.TableAccess {
			tables = append(tables, tableName)
			for operation := range tableAccess.Operations {
				operations = append(operations, operation)
			}
		}
		
		fmt.Fprintf(writer, "%s,%s,%s,\"%s\",\"%s\"\n",
			funcEntry.FunctionName,
			funcEntry.PackageName,
			funcEntry.FileName,
			joinStrings(tables, ";"),
			joinStrings(operations, ";"),
		)
	}
	
	// Table view CSV
	fmt.Fprintf(writer, "\n# Table View\n")
	fmt.Fprintf(writer, "Table,Functions,Operations\n")
	
	for _, tableEntry := range report.Dependencies.TableView {
		var functions []string
		var operations []string
		
		for funcName := range tableEntry.AccessedBy {
			functions = append(functions, funcName)
		}
		
		for operation := range tableEntry.OperationSummary {
			operations = append(operations, operation)
		}
		
		fmt.Fprintf(writer, "%s,\"%s\",\"%s\"\n",
			tableEntry.TableName,
			joinStrings(functions, ";"),
			joinStrings(operations, ";"),
		)
	}
	
	return nil
}

// formatHTML formats the report as HTML
func (f *Formatter) formatHTML(report *types.AnalysisReport, writer io.Writer) error {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>SQLC Dependency Analysis Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 20px; margin-bottom: 20px; }
        .summary { background-color: #e8f4f8; padding: 15px; margin-bottom: 20px; }
        .section { margin-bottom: 30px; }
        .section h2 { color: #333; border-bottom: 2px solid #ddd; padding-bottom: 10px; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        .function-name { font-weight: bold; color: #0066cc; }
        .table-name { font-weight: bold; color: #cc6600; }
        .operation { display: inline-block; background-color: #e0e0e0; padding: 2px 6px; margin: 2px; border-radius: 3px; }
        .operation.SELECT { background-color: #d4edda; }
        .operation.INSERT { background-color: #cce5ff; }
        .operation.UPDATE { background-color: #fff3cd; }
        .operation.DELETE { background-color: #f8d7da; }
    </style>
</head>
<body>
    <div class="header">
        <h1>SQLC Dependency Analysis Report</h1>
        <p>Generated on: %s</p>
    </div>
    
    <div class="summary">
        <h2>Summary</h2>
        <p><strong>Functions:</strong> %d</p>
        <p><strong>Tables:</strong> %d</p>
        <p><strong>Total Operations:</strong> %d</p>
    </div>
`
	
	fmt.Fprintf(writer, html, time.Now().Format("2006-01-02 15:04:05"), 
		report.Summary.FunctionCount, 
		report.Summary.TableCount,
		sumOperations(report.Summary.OperationCounts))
	
	// Function view
	fmt.Fprintf(writer, `
    <div class="section">
        <h2>Function View</h2>
        <table>
            <tr>
                <th>Function</th>
                <th>Package</th>
                <th>File</th>
                <th>Tables</th>
                <th>Operations</th>
            </tr>
`)
	
	for _, funcEntry := range report.Dependencies.FunctionView {
		fmt.Fprintf(writer, `
            <tr>
                <td class="function-name">%s</td>
                <td>%s</td>
                <td>%s</td>
                <td>%s</td>
                <td>%s</td>
            </tr>
`,
			funcEntry.FunctionName,
			funcEntry.PackageName,
			funcEntry.FileName,
			f.formatTableList(funcEntry.TableAccess),
			f.formatOperationList(funcEntry.TableAccess),
		)
	}
	
	fmt.Fprintf(writer, `
        </table>
    </div>
`)
	
	// Table view
	fmt.Fprintf(writer, `
    <div class="section">
        <h2>Table View</h2>
        <table>
            <tr>
                <th>Table</th>
                <th>Functions</th>
                <th>Operations</th>
            </tr>
`)
	
	for _, tableEntry := range report.Dependencies.TableView {
		fmt.Fprintf(writer, `
            <tr>
                <td class="table-name">%s</td>
                <td>%s</td>
                <td>%s</td>
            </tr>
`,
			tableEntry.TableName,
			f.formatFunctionList(tableEntry.AccessedBy),
			f.formatOperationSummary(tableEntry.OperationSummary),
		)
	}
	
	fmt.Fprintf(writer, `
        </table>
    </div>
`)
	
	// Circular dependencies
	if len(report.Circular) > 0 {
		fmt.Fprintf(writer, `
    <div class="section">
        <h2>Circular Dependencies</h2>
        <ul>
`)
		for _, circular := range report.Circular {
			fmt.Fprintf(writer, `
            <li>%s: %s</li>
`, circular.Type, joinStrings(circular.Functions, " → "))
		}
		fmt.Fprintf(writer, `
        </ul>
    </div>
`)
	}
	
	// Optimization suggestions
	if len(report.Suggestions) > 0 {
		fmt.Fprintf(writer, `
    <div class="section">
        <h2>Optimization Suggestions</h2>
        <ul>
`)
		for _, suggestion := range report.Suggestions {
			fmt.Fprintf(writer, `
            <li><strong>%s:</strong> %s</li>
`, suggestion.Type, suggestion.Description)
		}
		fmt.Fprintf(writer, `
        </ul>
    </div>
`)
	}
	
	fmt.Fprintf(writer, `
</body>
</html>
`)
	
	return nil
}

// Helper functions

func joinStrings(strs []string, separator string) string {
	if len(strs) == 0 {
		return ""
	}
	
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += separator + strs[i]
	}
	return result
}

func sumOperations(operationCounts map[string]int) int {
	total := 0
	for _, count := range operationCounts {
		total += count
	}
	return total
}

func (f *Formatter) formatTableList(tableAccess map[string]types.TableAccessInfo) string {
	var tables []string
	for tableName := range tableAccess {
		tables = append(tables, tableName)
	}
	return joinStrings(tables, ", ")
}

func (f *Formatter) formatOperationList(tableAccess map[string]types.TableAccessInfo) string {
	var operations []string
	for _, accessInfo := range tableAccess {
		for operation := range accessInfo.Operations {
			operations = append(operations, operation)
		}
	}
	return joinStrings(operations, ", ")
}

func (f *Formatter) formatFunctionList(accessedBy map[string]types.FunctionAccess) string {
	var functions []string
	for funcName := range accessedBy {
		functions = append(functions, funcName)
	}
	return joinStrings(functions, ", ")
}

func (f *Formatter) formatOperationSummary(operationSummary map[string]int) string {
	var operations []string
	for operation, count := range operationSummary {
		operations = append(operations, fmt.Sprintf("%s(%d)", operation, count))
	}
	return joinStrings(operations, ", ")
}