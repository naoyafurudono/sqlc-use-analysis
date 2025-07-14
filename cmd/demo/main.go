package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
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

func main() {
	fmt.Printf("%s%s=== SQLC Use Analysis Demo ===%s\n\n", colorBold, colorCyan, colorReset)
	
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
	
	fmt.Printf("%sAnalyzing sample e-commerce project...%s\n", colorYellow, colorReset)
	fmt.Printf("Project location: %s\n\n", fixturesPath)
	
	// デモ実行
	if err := runDemo(fixturesPath); err != nil {
		log.Fatalf("Demo failed: %v", err)
	}
}

func runDemo(projectPath string) error {
	// アナライザーを作成
	a := analyzer.New()
	
	fmt.Printf("%s1. Setting up analysis...%s\n", colorBlue, colorReset)
	
	// SQLクエリを定義（実際のプロジェクトのクエリ）
	queries := []analyzer.Query{
		{
			Name: "GetUser",
			SQL:  "SELECT id, name, email, created_at FROM users WHERE id = $1",
		},
		{
			Name: "ListUsers",
			SQL:  "SELECT id, name, email, created_at FROM users ORDER BY created_at DESC",
		},
		{
			Name: "CreateUser",
			SQL:  "INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, name, email, created_at",
		},
		{
			Name: "GetPost",
			SQL:  "SELECT p.id, p.title, p.content, p.author_id, p.created_at, u.name as author_name FROM posts p JOIN users u ON p.author_id = u.id WHERE p.id = $1",
		},
		{
			Name: "ListPostsByUser",
			SQL:  "SELECT id, title, content, author_id, created_at FROM posts WHERE author_id = $1 ORDER BY created_at DESC",
		},
		{
			Name: "CreatePost",
			SQL:  "INSERT INTO posts (title, content, author_id) VALUES ($1, $2, $3) RETURNING id, title, content, author_id, created_at",
		},
		{
			Name: "GetCommentsByPost",
			SQL:  "SELECT c.id, c.content, c.author_id, c.created_at, u.name as author_name FROM comments c JOIN users u ON c.author_id = u.id WHERE c.post_id = $1 ORDER BY c.created_at",
		},
		{
			Name: "CreateComment",
			SQL:  "INSERT INTO comments (post_id, author_id, content) VALUES ($1, $2, $3) RETURNING id, post_id, author_id, content, created_at",
		},
	}
	
	// Goパッケージのパスを定義
	goPackages := []string{
		filepath.Join(projectPath, "internal", "db"),
		filepath.Join(projectPath, "internal", "service"),
		filepath.Join(projectPath, "internal", "handler"),
	}
	
	fmt.Printf("  • SQL Queries: %d\n", len(queries))
	fmt.Printf("  • Go Packages: %d\n", len(goPackages))
	
	// 分析リクエストを作成
	request := analyzer.AnalysisRequest{
		SQLQueries:   queries,
		GoPackages:   goPackages,
		OutputFormat: "json",
		PrettyPrint:  true,
	}
	
	fmt.Printf("\n%s2. Running dependency analysis...%s\n", colorBlue, colorReset)
	
	// 分析実行（時間測定付き）
	start := time.Now()
	ctx := context.Background()
	result, err := a.Analyze(ctx, request)
	duration := time.Since(start)
	
	if err != nil {
		fmt.Printf("%sAnalysis failed: %v%s\n", colorRed, err, colorReset)
		
		// エラー詳細を表示
		errors := a.GetErrors()
		if len(errors) > 0 {
			fmt.Printf("\n%sError Details:%s\n", colorRed, colorReset)
			for i, err := range errors {
				if i >= 5 { // 最初の5個のエラーのみ表示
					fmt.Printf("  ... and %d more errors\n", len(errors)-5)
					break
				}
				fmt.Printf("  [%s] %s: %s\n", err.Severity, err.Category, err.Message)
			}
		}
		return err
	}
	
	fmt.Printf("  • Analysis completed in %v\n", duration)
	
	// 結果を表示
	fmt.Printf("\n%s3. Analysis Results%s\n", colorBlue, colorReset)
	displayResults(result)
	
	// 詳細な依存関係マップを表示
	fmt.Printf("\n%s4. Dependency Analysis%s\n", colorBlue, colorReset)
	displayDependencyAnalysis(result)
	
	// JSONファイルに結果を保存
	fmt.Printf("\n%s5. Saving detailed results...%s\n", colorBlue, colorReset)
	if err := saveResults(result); err != nil {
		return fmt.Errorf("failed to save results: %w", err)
	}
	
	fmt.Printf("\n%s%sDemo completed successfully!%s\n", colorBold, colorGreen, colorReset)
	fmt.Printf("Check './demo_results.json' for detailed analysis results.\n")
	
	return nil
}

func displayResults(result *analyzer.Result) {
	fmt.Printf("  • Functions analyzed: %s%d%s\n", colorGreen, result.Summary.FunctionCount, colorReset)
	fmt.Printf("  • Tables identified: %s%d%s\n", colorGreen, result.Summary.TableCount, colorReset)
	fmt.Printf("  • Dependencies found: %s%d%s\n", colorGreen, result.Summary.DependencyCount, colorReset)
	
	// テーブル一覧
	fmt.Printf("\n  %sTables:%s\n", colorPurple, colorReset)
	for tableName, tableInfo := range result.Tables {
		fmt.Printf("    • %s%s%s (accessed by %d functions)\n", 
			colorWhite, tableName, colorReset, len(tableInfo.AccessedBy))
	}
	
	// 操作統計
	if len(result.Summary.OperationCounts) > 0 {
		fmt.Printf("\n  %sOperations:%s\n", colorPurple, colorReset)
		for operation, count := range result.Summary.OperationCounts {
			fmt.Printf("    • %s: %s%d%s times\n", operation, colorWhite, count, colorReset)
		}
	}
}

func displayDependencyAnalysis(result *analyzer.Result) {
	// サービス層の関数に焦点を当てる
	fmt.Printf("  %sService Layer Analysis:%s\n", colorPurple, colorReset)
	
	serviceCount := 0
	for funcName, funcInfo := range result.Functions {
		if funcInfo.Package == "service" {
			serviceCount++
			fmt.Printf("    • %s%s%s:\n", colorWhite, funcName, colorReset)
			
			if len(funcInfo.TableAccess) == 0 {
				fmt.Printf("      - No direct table access\n")
			} else {
				for tableName, access := range funcInfo.TableAccess {
					fmt.Printf("      - %s%s%s: %v (%d calls)\n", 
						colorCyan, tableName, colorReset, access.Operations, access.Count)
				}
			}
		}
	}
	
	if serviceCount == 0 {
		fmt.Printf("    No service layer functions found\n")
	}
	
	// 複雑な依存関係（複数テーブルにアクセスする関数）
	fmt.Printf("\n  %sComplex Dependencies:%s\n", colorPurple, colorReset)
	complexFound := false
	
	for funcName, funcInfo := range result.Functions {
		if len(funcInfo.TableAccess) > 1 {
			complexFound = true
			tableNames := make([]string, 0, len(funcInfo.TableAccess))
			for tableName := range funcInfo.TableAccess {
				tableNames = append(tableNames, tableName)
			}
			fmt.Printf("    • %s%s%s accesses: %v\n", 
				colorWhite, funcName, colorReset, tableNames)
		}
	}
	
	if !complexFound {
		fmt.Printf("    No complex multi-table dependencies found\n")
	}
}

func saveResults(result *analyzer.Result) error {
	// JSON形式で結果を保存
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}
	
	filename := "demo_results.json"
	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	fmt.Printf("  • Results saved to %s%s%s (%d bytes)\n", 
		colorGreen, filename, colorReset, len(jsonData))
	
	return nil
}