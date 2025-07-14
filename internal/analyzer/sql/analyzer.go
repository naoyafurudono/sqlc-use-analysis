package sql

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/naoyafurudono/sqlc-use-analysis/internal/errors"
	"github.com/naoyafurudono/sqlc-use-analysis/pkg/types"
)

// Analyzer analyzes SQL queries and extracts table operations
type Analyzer struct {
	dialect         string
	caseSensitive   bool
	errorCollector  *errors.ErrorCollector
}

// NewAnalyzer creates a new SQL analyzer
func NewAnalyzer(dialect string, caseSensitive bool, errorCollector *errors.ErrorCollector) *Analyzer {
	return &Analyzer{
		dialect:        dialect,
		caseSensitive:  caseSensitive,
		errorCollector: errorCollector,
	}
}

// Query represents a SQL query from sqlc
type Query struct {
	Text     string `json:"text"`
	Name     string `json:"name"`
	Cmd      string `json:"cmd"`
	Filename string `json:"filename"`
}

// AnalyzeQueries analyzes multiple SQL queries
func (a *Analyzer) AnalyzeQueries(queries []Query) (map[string]types.SQLMethodInfo, error) {
	results := make(map[string]types.SQLMethodInfo)
	
	for _, query := range queries {
		methodInfo, err := a.AnalyzeQuery(query)
		if err != nil {
			// エラーを収集して処理を継続
			sqlErr := errors.NewError(errors.CategoryParse, errors.SeverityError, 
				fmt.Sprintf("failed to analyze query '%s': %v", query.Name, err))
			sqlErr.Details["query_name"] = query.Name
			sqlErr.Details["query_text"] = query.Text
			sqlErr.Details["filename"] = query.Filename
			
			if collectErr := a.errorCollector.Add(sqlErr); collectErr != nil {
				return results, collectErr
			}
			continue
		}
		
		results[methodInfo.MethodName] = methodInfo
	}
	
	return results, nil
}

// AnalyzeQuery analyzes a single SQL query
func (a *Analyzer) AnalyzeQuery(query Query) (types.SQLMethodInfo, error) {
	// メソッド名の生成
	methodName := a.generateMethodName(query.Name, query.Cmd)
	
	// SQL操作種別の判定
	operation, err := a.detectOperationType(query.Text)
	if err != nil {
		return types.SQLMethodInfo{}, fmt.Errorf("failed to detect operation type: %w", err)
	}
	
	// テーブル名の抽出
	tables, err := a.extractTables(query.Text, operation)
	if err != nil {
		return types.SQLMethodInfo{}, fmt.Errorf("failed to extract tables: %w", err)
	}
	
	// 結果の構築
	tableOps := make([]types.TableOperation, 0, len(tables))
	for _, table := range tables {
		tableOp := types.TableOperation{
			TableName:  table,
			Operations: []string{string(operation)},
		}
		tableOps = append(tableOps, tableOp)
	}
	
	return types.SQLMethodInfo{
		MethodName: methodName,
		Tables:     tableOps,
	}, nil
}

// generateMethodName generates a Go method name from query name and command
func (a *Analyzer) generateMethodName(queryName, cmd string) string {
	// クエリ名をPascalCaseに変換
	methodName := toPascalCase(queryName)
	
	// コマンドタイプに応じた調整
	switch cmd {
	case ":many":
		// 複数形にする場合の処理
		if !strings.HasSuffix(methodName, "s") && 
		   !strings.HasSuffix(methodName, "List") {
			// 簡単な複数形化（実際にはより複雑なルールが必要）
			if strings.HasSuffix(methodName, "y") {
				methodName = methodName[:len(methodName)-1] + "ies"
			} else {
				methodName = methodName + "s"
			}
		}
	}
	
	return methodName
}

// detectOperationType detects the SQL operation type
func (a *Analyzer) detectOperationType(sqlText string) (types.Operation, error) {
	// SQL文を正規化（改行、余分な空白を除去）
	normalizedSQL := normalizeSQL(sqlText)
	upperSQL := strings.ToUpper(strings.TrimSpace(normalizedSQL))
	
	switch {
	case strings.HasPrefix(upperSQL, "SELECT"):
		return types.OpSelect, nil
	case strings.HasPrefix(upperSQL, "INSERT"):
		return types.OpInsert, nil
	case strings.HasPrefix(upperSQL, "UPDATE"):
		return types.OpUpdate, nil
	case strings.HasPrefix(upperSQL, "DELETE"):
		return types.OpDelete, nil
	case strings.HasPrefix(upperSQL, "WITH"):
		// CTE（Common Table Expression）の場合は本体を解析
		return a.detectCTEOperationType(upperSQL)
	default:
		return "", fmt.Errorf("unknown SQL operation in: %s", sqlText)
	}
}

// detectCTEOperationType detects operation type in CTE
func (a *Analyzer) detectCTEOperationType(sqlText string) (types.Operation, error) {
	// WITH句の後の最終的なクエリを見つける
	// 簡単な実装：最後のSELECT/INSERT/UPDATE/DELETEを探す
	ctePattern := regexp.MustCompile(`(?i)WITH\s+.*?\)\s*(SELECT|INSERT|UPDATE|DELETE)`)
	matches := ctePattern.FindStringSubmatch(sqlText)
	
	if len(matches) >= 2 {
		switch strings.ToUpper(matches[1]) {
		case "SELECT":
			return types.OpSelect, nil
		case "INSERT":
			return types.OpInsert, nil
		case "UPDATE":
			return types.OpUpdate, nil
		case "DELETE":
			return types.OpDelete, nil
		}
	}
	
	// デフォルトではSELECTと仮定
	return types.OpSelect, nil
}

// extractTables extracts table names from SQL
func (a *Analyzer) extractTables(sqlText string, operation types.Operation) ([]string, error) {
	normalizedSQL := normalizeSQL(sqlText)
	
	var tables []string
	var err error
	
	switch operation {
	case types.OpSelect:
		tables, err = a.extractTablesFromSelect(normalizedSQL)
	case types.OpInsert:
		tables, err = a.extractTablesFromInsert(normalizedSQL)
	case types.OpUpdate:
		tables, err = a.extractTablesFromUpdate(normalizedSQL)
	case types.OpDelete:
		tables, err = a.extractTablesFromDelete(normalizedSQL)
	default:
		return nil, fmt.Errorf("unsupported operation: %v", operation)
	}
	
	if err != nil {
		return nil, err
	}
	
	// 重複を除去
	return removeDuplicates(tables), nil
}

// normalizeSQL normalizes SQL text
func normalizeSQL(sql string) string {
	// 改行を空白に変換
	sql = regexp.MustCompile(`\s+`).ReplaceAllString(sql, " ")
	// 前後の空白を除去
	return strings.TrimSpace(sql)
}

// toPascalCase converts string to PascalCase
func toPascalCase(s string) string {
	if s == "" {
		return s
	}
	
	// 既にPascalCaseの場合はそのまま返す
	if isPascalCase(s) {
		return s
	}
	
	// アンダースコアやハイフンで分割
	words := regexp.MustCompile(`[_\-\s]+`).Split(s, -1)
	result := ""
	
	for _, word := range words {
		if len(word) > 0 {
			// 最初の文字を大文字に、残りを小文字に
			result += strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	
	return result
}

// isPascalCase checks if a string is already in PascalCase format
func isPascalCase(s string) bool {
	if len(s) == 0 {
		return false
	}
	
	// 最初の文字が大文字かチェック
	if s[0] < 'A' || s[0] > 'Z' {
		return false
	}
	
	// アンダースコアやハイフンがないかチェック
	return !strings.ContainsAny(s, "_-")
}

// removeDuplicates removes duplicate strings from slice
func removeDuplicates(strs []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	
	for _, str := range strs {
		if !seen[str] {
			seen[str] = true
			result = append(result, str)
		}
	}
	
	return result
}