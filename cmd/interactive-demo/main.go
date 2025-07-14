package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/naoyafurudono/sqlc-use-analysis/pkg/analyzer"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorBold   = "\033[1m"
)

type DemoSession struct {
	analyzer     *analyzer.Analyzer
	result       *analyzer.Result
	projectPath  string
	scanner      *bufio.Scanner
}

func main() {
	fmt.Printf("%s%s=== Interactive SQLC Use Analysis Demo ===%s\n\n", colorBold, colorCyan, colorReset)
	
	// プロジェクトのルートパスを取得
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}
	
	// テストフィクスチャのパスを構築
	fixturesPath := filepath.Join(workDir, "test", "fixtures", "simple_project")
	
	// フィクスチャが存在するかチェック
	if _, err := os.Stat(fixturesPath); os.IsNotExist(err) {
		log.Fatalf("Demo fixture not found at %s. Please ensure you're running from the project root.", fixturesPath)
	}
	
	session := &DemoSession{
		analyzer:    analyzer.New(),
		projectPath: fixturesPath,
		scanner:     bufio.NewScanner(os.Stdin),
	}
	
	// デモセッション開始
	session.run()
}

func (d *DemoSession) run() {
	d.showWelcome()
	
	for {
		d.showMenu()
		choice := d.getInput("Select an option (1-8, q to quit): ")
		
		switch strings.ToLower(choice) {
		case "1":
			d.runBasicAnalysis()
		case "2":
			d.showProjectStructure()
		case "3":
			d.showSQLQueries()
		case "4":
			d.showDependencyGraph()
		case "5":
			d.showTableAnalysis()
		case "6":
			d.showFunctionAnalysis()
		case "7":
			d.exportResults()
		case "8":
			d.showErrorAnalysis()
		case "q", "quit", "exit":
			fmt.Printf("\n%sThank you for using SQLC Use Analysis Demo!%s\n", colorGreen, colorReset)
			return
		default:
			fmt.Printf("%sInvalid option. Please try again.%s\n", colorRed, colorReset)
		}
		
		fmt.Println() // 空行を追加
	}
}

func (d *DemoSession) showWelcome() {
	fmt.Printf("This interactive demo showcases the SQLC dependency analysis capabilities.\n")
	fmt.Printf("We'll analyze a sample e-commerce project with users, posts, and comments.\n\n")
	fmt.Printf("Project location: %s%s%s\n\n", colorCyan, d.projectPath, colorReset)
}

func (d *DemoSession) showMenu() {
	fmt.Printf("%s--- Demo Menu ---%s\n", colorBold, colorReset)
	fmt.Printf("1. %sRun Basic Analysis%s\n", colorGreen, colorReset)
	fmt.Printf("2. %sShow Project Structure%s\n", colorBlue, colorReset)
	fmt.Printf("3. %sShow SQL Queries%s\n", colorYellow, colorReset)
	fmt.Printf("4. %sShow Dependency Graph%s\n", colorPurple, colorReset)
	fmt.Printf("5. %sShow Table Analysis%s\n", colorCyan, colorReset)
	fmt.Printf("6. %sShow Function Analysis%s\n", colorWhite, colorReset)
	fmt.Printf("7. %sExport Results%s\n", colorGreen, colorReset)
	fmt.Printf("8. %sShow Error Analysis%s\n", colorRed, colorReset)
	fmt.Printf("q. %sQuit%s\n", colorRed, colorReset)
	fmt.Println()
}

func (d *DemoSession) getInput(prompt string) string {
	fmt.Print(prompt)
	d.scanner.Scan()
	return strings.TrimSpace(d.scanner.Text())
}

func (d *DemoSession) runBasicAnalysis() {
	fmt.Printf("%s=== Running Basic Analysis ===%s\n\n", colorBold, colorBlue, colorReset)
	
	if d.result != nil {
		fmt.Printf("Analysis already completed. Showing cached results...\n\n")
		d.displayBasicResults()
		return
	}
	
	// SQLクエリを定義
	queries := []analyzer.Query{
		{Name: "GetUser", SQL: "SELECT id, name, email, created_at FROM users WHERE id = $1"},
		{Name: "ListUsers", SQL: "SELECT id, name, email, created_at FROM users ORDER BY created_at DESC"},
		{Name: "CreateUser", SQL: "INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, name, email, created_at"},
		{Name: "GetPost", SQL: "SELECT p.id, p.title, p.content, p.author_id, p.created_at, u.name as author_name FROM posts p JOIN users u ON p.author_id = u.id WHERE p.id = $1"},
		{Name: "ListPostsByUser", SQL: "SELECT id, title, content, author_id, created_at FROM posts WHERE author_id = $1 ORDER BY created_at DESC"},
		{Name: "CreatePost", SQL: "INSERT INTO posts (title, content, author_id) VALUES ($1, $2, $3) RETURNING id, title, content, author_id, created_at"},
		{Name: "GetCommentsByPost", SQL: "SELECT c.id, c.content, c.author_id, c.created_at, u.name as author_name FROM comments c JOIN users u ON c.author_id = u.id WHERE c.post_id = $1 ORDER BY c.created_at"},
		{Name: "CreateComment", SQL: "INSERT INTO comments (post_id, author_id, content) VALUES ($1, $2, $3) RETURNING id, post_id, author_id, content, created_at"},
	}
	
	// Goパッケージのパスを定義
	goPackages := []string{
		filepath.Join(d.projectPath, "internal", "db"),
		filepath.Join(d.projectPath, "internal", "service"),
		filepath.Join(d.projectPath, "internal", "handler"),
	}
	
	request := analyzer.AnalysisRequest{
		SQLQueries:   queries,
		GoPackages:   goPackages,
		OutputFormat: "json",
		PrettyPrint:  true,
	}
	
	fmt.Printf("Analyzing %d SQL queries across %d Go packages...\n", len(queries), len(goPackages))
	
	// 分析実行
	start := time.Now()
	ctx := context.Background()
	result, err := d.analyzer.Analyze(ctx, request)
	duration := time.Since(start)
	
	if err != nil {
		fmt.Printf("%sAnalysis failed: %v%s\n", colorRed, err, colorReset)
		d.showAnalysisErrors()
		return
	}
	
	d.result = result
	fmt.Printf("%sAnalysis completed in %v%s\n\n", colorGreen, duration, colorReset)
	
	d.displayBasicResults()
}

func (d *DemoSession) displayBasicResults() {
	if d.result == nil {
		fmt.Printf("%sNo analysis results available. Please run basic analysis first.%s\n", colorRed, colorReset)
		return
	}
	
	r := d.result
	fmt.Printf("%sAnalysis Summary:%s\n", colorBold, colorReset)
	fmt.Printf("• Functions: %s%d%s\n", colorGreen, r.Summary.FunctionCount, colorReset)
	fmt.Printf("• Tables: %s%d%s\n", colorGreen, r.Summary.TableCount, colorReset)
	fmt.Printf("• Dependencies: %s%d%s\n", colorGreen, r.Summary.DependencyCount, colorReset)
	
	if len(r.Summary.OperationCounts) > 0 {
		fmt.Printf("\n%sOperation Distribution:%s\n", colorPurple, colorReset)
		for op, count := range r.Summary.OperationCounts {
			fmt.Printf("• %s: %s%d%s\n", op, colorWhite, count, colorReset)
		}
	}
}

func (d *DemoSession) showProjectStructure() {
	fmt.Printf("%s=== Project Structure ===%s\n\n", colorBold, colorBlue, colorReset)
	
	structure := map[string][]string{
		"Database Layer (internal/db)": {
			"models.go - Database models and types",
			"query.sql.go - SQLC generated query functions",
		},
		"Service Layer (internal/service)": {
			"user_service.go - User business logic",
			"post_service.go - Post business logic",
		},
		"Handler Layer (internal/handler)": {
			"user_handler.go - User HTTP handlers",
			"post_handler.go - Post HTTP handlers",
		},
	}
	
	for layer, files := range structure {
		fmt.Printf("%s%s:%s\n", colorPurple, layer, colorReset)
		for _, file := range files {
			fmt.Printf("  • %s\n", file)
		}
		fmt.Println()
	}
}

func (d *DemoSession) showSQLQueries() {
	fmt.Printf("%s=== SQL Queries ===%s\n\n", colorBold, colorBlue, colorReset)
	
	queries := map[string]string{
		"GetUser":           "SELECT id, name, email, created_at FROM users WHERE id = $1",
		"ListUsers":         "SELECT id, name, email, created_at FROM users ORDER BY created_at DESC",
		"CreateUser":        "INSERT INTO users (name, email) VALUES ($1, $2) RETURNING *",
		"GetPost":           "SELECT p.*, u.name as author_name FROM posts p JOIN users u ON p.author_id = u.id WHERE p.id = $1",
		"ListPostsByUser":   "SELECT * FROM posts WHERE author_id = $1 ORDER BY created_at DESC",
		"CreatePost":        "INSERT INTO posts (title, content, author_id) VALUES ($1, $2, $3) RETURNING *",
		"GetCommentsByPost": "SELECT c.*, u.name as author_name FROM comments c JOIN users u ON c.author_id = u.id WHERE c.post_id = $1",
		"CreateComment":     "INSERT INTO comments (post_id, author_id, content) VALUES ($1, $2, $3) RETURNING *",
	}
	
	i := 1
	for name, sql := range queries {
		fmt.Printf("%s%d. %s:%s\n", colorGreen, i, name, colorReset)
		fmt.Printf("   %s\n\n", sql)
		i++
	}
}

func (d *DemoSession) showDependencyGraph() {
	fmt.Printf("%s=== Dependency Graph ===%s\n\n", colorBold, colorBlue, colorReset)
	
	if d.result == nil {
		fmt.Printf("%sPlease run basic analysis first.%s\n", colorRed, colorReset)
		return
	}
	
	// 層別に依存関係を表示
	layers := []string{"handler", "service", "db"}
	
	for _, layer := range layers {
		fmt.Printf("%s%s Layer:%s\n", colorPurple, strings.Title(layer), colorReset)
		
		found := false
		for funcName, funcInfo := range d.result.Functions {
			if funcInfo.Package == layer {
				found = true
				fmt.Printf("  • %s%s%s\n", colorWhite, funcName, colorReset)
				
				if len(funcInfo.TableAccess) > 0 {
					for tableName, access := range funcInfo.TableAccess {
						fmt.Printf("    └─ %s%s%s: %v\n", colorCyan, tableName, colorReset, access.Operations)
					}
				} else {
					fmt.Printf("    └─ %sNo direct database access%s\n", colorYellow, colorReset)
				}
			}
		}
		
		if !found {
			fmt.Printf("  %sNo functions found%s\n", colorYellow, colorReset)
		}
		fmt.Println()
	}
}

func (d *DemoSession) showTableAnalysis() {
	fmt.Printf("%s=== Table Analysis ===%s\n\n", colorBold, colorBlue, colorReset)
	
	if d.result == nil {
		fmt.Printf("%sPlease run basic analysis first.%s\n", colorRed, colorReset)
		return
	}
	
	for tableName, tableInfo := range d.result.Tables {
		fmt.Printf("%s%s Table:%s\n", colorPurple, strings.Title(tableName), colorReset)
		fmt.Printf("  • Accessed by %s%d%s functions\n", colorGreen, len(tableInfo.AccessedBy), colorReset)
		
		if len(tableInfo.OperationCount) > 0 {
			fmt.Printf("  • Operations:\n")
			for op, count := range tableInfo.OperationCount {
				fmt.Printf("    - %s: %s%d%s times\n", op, colorWhite, count, colorReset)
			}
		}
		
		if len(tableInfo.AccessedBy) > 0 {
			fmt.Printf("  • Accessed by:\n")
			for _, funcName := range tableInfo.AccessedBy {
				fmt.Printf("    - %s\n", funcName)
			}
		}
		fmt.Println()
	}
}

func (d *DemoSession) showFunctionAnalysis() {
	fmt.Printf("%s=== Function Analysis ===%s\n\n", colorBold, colorBlue, colorReset)
	
	if d.result == nil {
		fmt.Printf("%sPlease run basic analysis first.%s\n", colorRed, colorReset)
		return
	}
	
	// 関数の種類別に分析
	categories := map[string][]string{
		"Database Functions": {},
		"Service Functions":  {},
		"Handler Functions":  {},
	}
	
	for funcName, funcInfo := range d.result.Functions {
		switch funcInfo.Package {
		case "db":
			categories["Database Functions"] = append(categories["Database Functions"], funcName)
		case "service":
			categories["Service Functions"] = append(categories["Service Functions"], funcName)
		case "handler":
			categories["Handler Functions"] = append(categories["Handler Functions"], funcName)
		}
	}
	
	for category, functions := range categories {
		if len(functions) > 0 {
			fmt.Printf("%s%s (%d):%s\n", colorPurple, category, len(functions), colorReset)
			for _, funcName := range functions {
				funcInfo := d.result.Functions[funcName]
				tableCount := len(funcInfo.TableAccess)
				if tableCount > 0 {
					fmt.Printf("  • %s - accesses %s%d%s tables\n", funcName, colorGreen, tableCount, colorReset)
				} else {
					fmt.Printf("  • %s - %sno table access%s\n", funcName, colorYellow, colorReset)
				}
			}
			fmt.Println()
		}
	}
}

func (d *DemoSession) exportResults() {
	fmt.Printf("%s=== Export Results ===%s\n\n", colorBold, colorBlue, colorReset)
	
	if d.result == nil {
		fmt.Printf("%sPlease run basic analysis first.%s\n", colorRed, colorReset)
		return
	}
	
	fmt.Printf("Available formats:\n")
	fmt.Printf("1. JSON (detailed)\n")
	fmt.Printf("2. JSON (summary only)\n")
	fmt.Printf("3. Text report\n")
	
	choice := d.getInput("Select format (1-3): ")
	
	var filename string
	var data []byte
	var err error
	
	switch choice {
	case "1":
		filename = "detailed_analysis.json"
		data, err = json.MarshalIndent(d.result, "", "  ")
	case "2":
		filename = "summary_analysis.json"
		summary := map[string]interface{}{
			"summary": d.result.Summary,
			"tables":  d.result.Tables,
		}
		data, err = json.MarshalIndent(summary, "", "  ")
	case "3":
		filename = "analysis_report.txt"
		data = []byte(d.generateTextReport())
	default:
		fmt.Printf("%sInvalid choice.%s\n", colorRed, colorReset)
		return
	}
	
	if err != nil {
		fmt.Printf("%sFailed to generate export data: %v%s\n", colorRed, err, colorReset)
		return
	}
	
	if err := os.WriteFile(filename, data, 0644); err != nil {
		fmt.Printf("%sFailed to write file: %v%s\n", colorRed, err, colorReset)
		return
	}
	
	fmt.Printf("%sResults exported to %s (%d bytes)%s\n", colorGreen, filename, len(data), colorReset)
}

func (d *DemoSession) generateTextReport() string {
	var report strings.Builder
	
	report.WriteString("SQLC Use Analysis Report\n")
	report.WriteString("========================\n\n")
	
	report.WriteString("Summary:\n")
	report.WriteString(fmt.Sprintf("- Functions: %d\n", d.result.Summary.FunctionCount))
	report.WriteString(fmt.Sprintf("- Tables: %d\n", d.result.Summary.TableCount))
	report.WriteString(fmt.Sprintf("- Dependencies: %d\n", d.result.Summary.DependencyCount))
	report.WriteString("\n")
	
	report.WriteString("Tables:\n")
	for tableName, tableInfo := range d.result.Tables {
		report.WriteString(fmt.Sprintf("- %s (accessed by %d functions)\n", tableName, len(tableInfo.AccessedBy)))
	}
	report.WriteString("\n")
	
	report.WriteString("Dependencies:\n")
	for _, dep := range d.result.Dependencies {
		report.WriteString(fmt.Sprintf("- %s -> %s (%s via %s)\n", dep.Function, dep.Table, dep.Operation, dep.Method))
	}
	
	return report.String()
}

func (d *DemoSession) showErrorAnalysis() {
	fmt.Printf("%s=== Error Analysis ===%s\n\n", colorBold, colorBlue, colorReset)
	
	errors := d.analyzer.GetErrors()
	if len(errors) == 0 {
		fmt.Printf("%sNo errors found during analysis!%s\n", colorGreen, colorReset)
		return
	}
	
	fmt.Printf("Found %s%d%s errors:\n\n", colorRed, len(errors), colorReset)
	
	for i, err := range errors {
		if i >= 10 {
			fmt.Printf("... and %d more errors\n", len(errors)-10)
			break
		}
		
		fmt.Printf("%s%d. [%s] %s:%s\n", colorRed, i+1, err.Severity, err.Category, colorReset)
		fmt.Printf("   %s\n", err.Message)
		if len(err.Details) > 0 {
			fmt.Printf("   Details: %v\n", err.Details)
		}
		fmt.Println()
	}
}

func (d *DemoSession) showAnalysisErrors() {
	errors := d.analyzer.GetErrors()
	if len(errors) == 0 {
		return
	}
	
	fmt.Printf("\n%sAnalysis Errors (%d):%s\n", colorRed, len(errors), colorReset)
	for i, err := range errors {
		if i >= 5 {
			fmt.Printf("... and %d more errors\n", len(errors)-5)
			break
		}
		fmt.Printf("  [%s] %s: %s\n", err.Severity, err.Category, err.Message)
	}
}