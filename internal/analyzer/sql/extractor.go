package sql

import (
	"fmt"
	"regexp"
	"strings"
)

// extractTablesFromSelect extracts table names from SELECT statements
func (a *Analyzer) extractTablesFromSelect(sqlText string) ([]string, error) {
	var tables []string
	
	// FROM句のテーブルを抽出
	fromTables, err := a.extractFromClause(sqlText)
	if err != nil {
		return nil, fmt.Errorf("failed to extract FROM clause: %w", err)
	}
	tables = append(tables, fromTables...)
	
	// JOIN句のテーブルを抽出
	joinTables, err := a.extractJoinTables(sqlText)
	if err != nil {
		return nil, fmt.Errorf("failed to extract JOIN tables: %w", err)
	}
	tables = append(tables, joinTables...)
	
	return tables, nil
}

// extractTablesFromInsert extracts table names from INSERT statements
func (a *Analyzer) extractTablesFromInsert(sqlText string) ([]string, error) {
	// MySQL/PostgreSQL共通: INSERT INTO table_name の形式
	// また、バッククォートでのテーブル名も対応
	pattern := regexp.MustCompile(`(?i)INSERT\s+INTO\s+` + a.getTableNamePattern())
	matches := pattern.FindStringSubmatch(sqlText)
	
	if len(matches) >= 2 {
		tableName := a.normalizeTableName(matches[1])
		return []string{tableName}, nil
	}
	
	return nil, fmt.Errorf("could not extract table name from INSERT statement: %s", sqlText)
}

// extractTablesFromUpdate extracts table names from UPDATE statements
func (a *Analyzer) extractTablesFromUpdate(sqlText string) ([]string, error) {
	var tables []string
	
	// UPDATE table_name SET の形式（MySQL/PostgreSQL対応）
	pattern := regexp.MustCompile(`(?i)UPDATE\s+` + a.getTableNamePattern() + `\s+SET`)
	matches := pattern.FindStringSubmatch(sqlText)
	
	if len(matches) >= 2 {
		tableName := a.normalizeTableName(matches[1])
		tables = append(tables, tableName)
	}
	
	// FROM句がある場合のテーブルも抽出
	if strings.Contains(strings.ToUpper(sqlText), " FROM ") {
		fromTables, err := a.extractFromClause(sqlText)
		if err == nil {
			tables = append(tables, fromTables...)
		}
	}
	
	// JOIN句のテーブルも抽出
	joinTables, err := a.extractJoinTables(sqlText)
	if err == nil {
		tables = append(tables, joinTables...)
	}
	
	if len(tables) == 0 {
		return nil, fmt.Errorf("could not extract table name from UPDATE statement: %s", sqlText)
	}
	
	// 重複を除去
	return removeDuplicates(tables), nil
}

// extractTablesFromDelete extracts table names from DELETE statements
func (a *Analyzer) extractTablesFromDelete(sqlText string) ([]string, error) {
	var tables []string
	
	// DELETE FROM table_name の形式（MySQL/PostgreSQL対応）
	pattern := regexp.MustCompile(`(?i)DELETE\s+FROM\s+` + a.getTableNamePattern())
	matches := pattern.FindStringSubmatch(sqlText)
	
	if len(matches) >= 2 {
		tableName := a.normalizeTableName(matches[1])
		tables = append(tables, tableName)
	}
	
	// USING句がある場合のテーブルも抽出
	if strings.Contains(strings.ToUpper(sqlText), " USING ") {
		usingTables, err := a.extractUsingClause(sqlText)
		if err == nil {
			tables = append(tables, usingTables...)
		}
	}
	
	// JOIN句のテーブルも抽出
	joinTables, err := a.extractJoinTables(sqlText)
	if err == nil {
		tables = append(tables, joinTables...)
	}
	
	if len(tables) == 0 {
		return nil, fmt.Errorf("could not extract table name from DELETE statement: %s", sqlText)
	}
	
	return tables, nil
}

// extractFromClause extracts table names from FROM clause
func (a *Analyzer) extractFromClause(sqlText string) ([]string, error) {
	// よりシンプルなアプローチ: FROMの後で最初のキーワードまで
	fromPattern := regexp.MustCompile(`(?i)\bFROM\s+(.+?)(?:\s+(?:INNER|LEFT|RIGHT|FULL|CROSS|JOIN|WHERE|ORDER|GROUP|HAVING|LIMIT)|$)`)
	matches := fromPattern.FindStringSubmatch(sqlText)
	
	if len(matches) < 2 {
		return []string{}, nil
	}
	
	fromClause := strings.TrimSpace(matches[1])
	
	// JOINキーワードで終わっている場合は除去
	joinKeywords := []string{"INNER", "LEFT", "RIGHT", "FULL", "CROSS", "JOIN"}
	for _, keyword := range joinKeywords {
		pattern := regexp.MustCompile(`(?i)\s+` + keyword + `$`)
		fromClause = pattern.ReplaceAllString(fromClause, "")
	}
	
	return a.parseTableList(fromClause), nil
}

// extractJoinTables extracts table names from JOIN clauses
func (a *Analyzer) extractJoinTables(sqlText string) ([]string, error) {
	tableSet := make(map[string]bool)
	
	// 各種JOIN句のパターン（MySQL/PostgreSQL対応）
	tablePattern := a.getTableNamePattern()
	joinPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bINNER\s+JOIN\s+` + tablePattern),
		regexp.MustCompile(`(?i)\bLEFT\s+(?:OUTER\s+)?JOIN\s+` + tablePattern),
		regexp.MustCompile(`(?i)\bRIGHT\s+(?:OUTER\s+)?JOIN\s+` + tablePattern),
		regexp.MustCompile(`(?i)\bFULL\s+(?:OUTER\s+)?JOIN\s+` + tablePattern),
		regexp.MustCompile(`(?i)\bCROSS\s+JOIN\s+` + tablePattern),
		regexp.MustCompile(`(?i)\bJOIN\s+` + tablePattern), // 単純なJOIN
	}
	
	for _, pattern := range joinPatterns {
		matches := pattern.FindAllStringSubmatch(sqlText, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				tableName := a.normalizeTableName(match[1])
				tableSet[tableName] = true
			}
		}
	}
	
	// セットからスライスに変換
	var tables []string
	for table := range tableSet {
		tables = append(tables, table)
	}
	
	return tables, nil
}

// extractUsingClause extracts table names from USING clause (DELETE ... USING ...)
func (a *Analyzer) extractUsingClause(sqlText string) ([]string, error) {
	pattern := regexp.MustCompile(`(?i)\bUSING\s+(.+?)(?:\s+WHERE|\s+ORDER|\s+GROUP|\s+HAVING|\s+LIMIT|$)`)
	matches := pattern.FindStringSubmatch(sqlText)
	
	if len(matches) < 2 {
		return []string{}, nil
	}
	
	usingClause := strings.TrimSpace(matches[1])
	return a.parseTableList(usingClause), nil
}

// parseTableList parses a comma-separated list of tables
func (a *Analyzer) parseTableList(tableList string) []string {
	var tables []string
	
	// カンマで分割
	parts := strings.Split(tableList, ",")
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		
		// サブクエリの場合はスキップ
		if strings.Contains(part, "(") {
			continue
		}
		
		// エイリアスを除去（table_name AS alias_name または table_name alias_name）
		aliasPattern := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*)\s+(?:AS\s+)?([a-zA-Z_][a-zA-Z0-9_]*)$`)
		if matches := aliasPattern.FindStringSubmatch(part); len(matches) >= 2 {
			tableName := a.normalizeTableName(matches[1])
			tables = append(tables, tableName)
		} else {
			// 単純なテーブル名の場合
			tablePattern := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*)`)
			if matches := tablePattern.FindStringSubmatch(part); len(matches) >= 2 {
				tableName := a.normalizeTableName(matches[1])
				tables = append(tables, tableName)
			}
		}
	}
	
	return tables
}

// normalizeTableName normalizes table name based on case sensitivity settings
func (a *Analyzer) normalizeTableName(tableName string) string {
	tableName = strings.TrimSpace(tableName)
	
	// MySQL/PostgreSQLのクォートを除去
	switch a.dialect {
	case "mysql":
		// バッククォートを除去
		tableName = strings.Trim(tableName, "`")
	case "postgresql":
		// ダブルクォートを除去
		tableName = strings.Trim(tableName, "\"")
	}
	
	if !a.caseSensitive {
		tableName = strings.ToLower(tableName)
	}
	
	return tableName
}

// isSubquery checks if the given text is a subquery
func (a *Analyzer) isSubquery(text string) bool {
	text = strings.TrimSpace(text)
	return strings.HasPrefix(text, "(") && strings.HasSuffix(text, ")")
}

// getTableNamePattern returns the regex pattern for table names based on dialect
func (a *Analyzer) getTableNamePattern() string {
	switch a.dialect {
	case "mysql":
		// MySQL: バッククォートでのテーブル名をサポート
		return `(` + "`" + `[a-zA-Z_][a-zA-Z0-9_]*` + "`" + `|[a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*)`
	case "postgresql":
		// PostgreSQL: ダブルクォートでのテーブル名をサポート
		return `("[a-zA-Z_][a-zA-Z0-9_]*"|[a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*)`
	default:
		// デフォルト（標準SQL）
		return `([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*)`
	}
}