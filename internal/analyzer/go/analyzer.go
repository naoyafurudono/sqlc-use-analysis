package gostatic

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
	"golang.org/x/tools/go/packages"

	"github.com/naoyafurudono/sqlc-use-analysis/internal/errors"
	pkgtypes "github.com/naoyafurudono/sqlc-use-analysis/pkg/types"
)

// Analyzer analyzes Go source code to extract function definitions and method calls
type Analyzer struct {
	packagePath     string
	errorCollector  *errors.ErrorCollector
	fset            *token.FileSet
	packages        []*packages.Package
}

// NewAnalyzer creates a new Go static analyzer
func NewAnalyzer(packagePath string, errorCollector *errors.ErrorCollector) *Analyzer {
	return &Analyzer{
		packagePath:    packagePath,
		errorCollector: errorCollector,
		fset:          token.NewFileSet(),
	}
}

// LoadPackages loads Go packages for analysis
func (a *Analyzer) LoadPackages(patterns ...string) error {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedSyntax |
			packages.NeedTypesInfo | packages.NeedTypesSizes,
		Fset: a.fset,
	}

	// Use error recovery for package loading
	err := errors.SafeExecute(a.errorCollector, func() error {
		pkgs, err := packages.Load(cfg, patterns...)
		if err != nil {
			return fmt.Errorf("failed to load packages: %w", err)
		}

		// Check for package loading errors
		for _, pkg := range pkgs {
			if len(pkg.Errors) > 0 {
				for _, pkgErr := range pkg.Errors {
					goErr := errors.NewError(errors.CategoryParse, errors.SeverityError,
						fmt.Sprintf("package loading error: %s", pkgErr.Msg))
					goErr.Details["package"] = pkg.PkgPath
					goErr.Details["package_name"] = pkg.Name
					goErr.Details["error_position"] = pkgErr.Pos

					if collectErr := a.errorCollector.Add(goErr); collectErr != nil {
						return collectErr
					}
				}
			}
		}

		a.packages = pkgs
		return nil
	}, "Go package loading")

	return err
}

// AnalyzePackages analyzes loaded packages and extracts function information
func (a *Analyzer) AnalyzePackages() (map[string]pkgtypes.GoFunctionInfo, error) {
	if len(a.packages) == 0 {
		return nil, fmt.Errorf("no packages loaded")
	}

	functions := make(map[string]pkgtypes.GoFunctionInfo)

	// Use error recovery for robust package processing
	partialResult := errors.ProcessWithPartialFailure(
		a.packages,
		func(pkg *packages.Package) error {
			pkgFunctions, err := a.analyzePackage(pkg)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("failed to analyze package '%s'", pkg.PkgPath))
			}

			// 関数情報をマージ
			for funcName, funcInfo := range pkgFunctions {
				functions[funcName] = funcInfo
			}
			return nil
		},
		a.errorCollector,
		"Go package analysis",
	)

	// Add package context to errors
	for _, err := range partialResult.Errors {
		for _, pkg := range a.packages {
			if strings.Contains(err.Message, pkg.PkgPath) {
				err.Details["package"] = pkg.PkgPath
				err.Details["package_name"] = pkg.Name
				break
			}
		}
	}

	return functions, nil
}

// analyzePackage analyzes a single package
func (a *Analyzer) analyzePackage(pkg *packages.Package) (map[string]pkgtypes.GoFunctionInfo, error) {
	functions := make(map[string]pkgtypes.GoFunctionInfo)

	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.FuncDecl:
				funcInfo, err := a.analyzeFuncDecl(node, pkg)
				if err != nil {
					// エラーを収集して処理を継続
					goErr := errors.NewError(errors.CategoryParse, errors.SeverityError,
						fmt.Sprintf("failed to analyze function '%s': %v", node.Name.Name, err))
					goErr.Details["function"] = node.Name.Name
					goErr.Details["package"] = pkg.PkgPath

					if collectErr := a.errorCollector.Add(goErr); collectErr != nil {
						return false
					}
					return true
				}

				functions[funcInfo.FunctionName] = funcInfo
			}
			return true
		})
	}

	return functions, nil
}

// analyzeFuncDecl analyzes a function declaration
func (a *Analyzer) analyzeFuncDecl(funcDecl *ast.FuncDecl, pkg *packages.Package) (pkgtypes.GoFunctionInfo, error) {
	funcName := funcDecl.Name.Name
	
	// レシーバーがある場合はメソッド名を調整
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		receiverType := a.extractReceiverType(funcDecl.Recv.List[0].Type)
		funcName = fmt.Sprintf("%s.%s", receiverType, funcName)
	}

	// 関数の位置情報を取得
	pos := a.fset.Position(funcDecl.Pos())

	funcInfo := pkgtypes.GoFunctionInfo{
		FunctionName: funcName,
		PackageName:  pkg.Name,
		FileName:     pos.Filename,
		FilePath:     pos.Filename,
		StartLine:    pos.Line,
		EndLine:      a.fset.Position(funcDecl.End()).Line,
		SQLCalls:     []pkgtypes.SQLCall{},
	}

	// 関数内のSQLメソッド呼び出しを抽出
	sqlCalls := a.extractSQLCalls(funcDecl.Body, pkg)
	funcInfo.SQLCalls = sqlCalls

	return funcInfo, nil
}

// extractReceiverType extracts receiver type name from receiver expression
func (a *Analyzer) extractReceiverType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return a.extractReceiverType(t.X)
	default:
		return "Unknown"
	}
}

// extractSQLCalls extracts SQL method calls from a function body
func (a *Analyzer) extractSQLCalls(body *ast.BlockStmt, pkg *packages.Package) []pkgtypes.SQLCall {
	var sqlCalls []pkgtypes.SQLCall

	if body == nil {
		return sqlCalls
	}

	ast.Inspect(body, func(n ast.Node) bool {
		if callExpr, ok := n.(*ast.CallExpr); ok {
			if sqlCall := a.analyzeSQLCall(callExpr, pkg); sqlCall != nil {
				sqlCalls = append(sqlCalls, *sqlCall)
			}
		}
		return true
	})

	return sqlCalls
}

// analyzeSQLCall analyzes a function call to determine if it's an SQL method call
func (a *Analyzer) analyzeSQLCall(callExpr *ast.CallExpr, pkg *packages.Package) *pkgtypes.SQLCall {
	// セレクター表現 (e.g., db.GetUser(), queries.ListUsers())
	if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		methodName := selExpr.Sel.Name
		
		// 型情報を使用して呼び出し元の型を判定
		if pkg.TypesInfo != nil {
			if objType := pkg.TypesInfo.TypeOf(selExpr.X); objType != nil {
				// SQLCで生成されたクエリメソッドかどうかを判定
				if a.isSQLCMethod(objType, methodName) {
					pos := a.fset.Position(callExpr.Pos())
					return &pkgtypes.SQLCall{
						MethodName: methodName,
						Line:       pos.Line,
						Column:     pos.Column,
					}
				}
			}
		}
	}

	return nil
}

// isSQLCMethod determines if a method call is an SQLC-generated query method
func (a *Analyzer) isSQLCMethod(objType types.Type, methodName string) bool {
	// 型名を取得
	typeName := objType.String()
	
	// まず、明らかにSQL driverメソッドを除外
	if a.isStandardSQLMethod(methodName) {
		return false
	}
	
	// SQLC生成のQueries型かチェック（より厳密に）
	if !a.isQueriesType(typeName) {
		return false
	}
	
	// メソッド名がsqlcパターンかチェック
	if a.isSQLCMethodName(methodName) {
		return true
	}
	
	return false
}

// isStandardSQLMethod checks if method name is a standard SQL driver method
func (a *Analyzer) isStandardSQLMethod(methodName string) bool {
	standardMethods := []string{
		"QueryRowContext", "QueryContext", "ExecContext", "PrepareContext",
		"Query", "QueryRow", "Exec", "Prepare",
		"Scan", "Close", "Next", "Err", "Columns", "ColumnTypes",
		"Begin", "Commit", "Rollback", "SetMaxIdleConns", "SetMaxOpenConns",
		"Ping", "Stats", "Driver",
	}
	
	for _, method := range standardMethods {
		if methodName == method {
			return true
		}
	}
	
	return false
}

// isQueriesType checks if type is an SQLC Queries type (more strict)
func (a *Analyzer) isQueriesType(typeName string) bool {
	// SQLC生成のQueries型の厳密なパターンチェック
	// 例: *github.com/example/db.Queries, *main.Queries, *db.Queries
	patterns := []string{
		".Queries",  // パッケージ.Queries
		"*Queries",  // *Queries
	}
	
	for _, pattern := range patterns {
		if contains(typeName, pattern) {
			return true
		}
	}
	
	return false
}

// isSQLCMethodName checks if method name follows SQLC patterns
func (a *Analyzer) isSQLCMethodName(methodName string) bool {
	// SQLC generated method names are typically PascalCase and not standard SQL methods
	if !a.isPascalCase(methodName) {
		return false
	}
	
	// Common SQLC method name patterns
	commonPrefixes := []string{
		"Get", "List", "Create", "Update", "Delete", "Count", "Find", "Select", "Insert",
	}
	
	for _, prefix := range commonPrefixes {
		if len(methodName) >= len(prefix) && methodName[:len(prefix)] == prefix {
			return true
		}
	}
	
	return false
}

// containsQueriesType checks if type name contains common SQLC patterns (deprecated)
func (a *Analyzer) containsQueriesType(typeName string) bool {
	// 一般的なSQLCの型パターン
	patterns := []string{
		"Queries",
		"queries",
	}
	
	for _, pattern := range patterns {
		if contains(typeName, pattern) {
			return true
		}
	}
	
	return false
}

// isPascalCase checks if a string is in PascalCase format
func (a *Analyzer) isPascalCase(s string) bool {
	if len(s) == 0 {
		return false
	}
	
	// 最初の文字が大文字かチェック
	first := s[0]
	return first >= 'A' && first <= 'Z'
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && 
		     (s[:len(substr)] == substr || 
		      s[len(s)-len(substr):] == substr ||
		      containsSubstring(s, substr))))
}

// containsSubstring checks if s contains substr anywhere
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}