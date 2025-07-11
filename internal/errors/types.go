package errors

import (
	"fmt"
	"runtime"
	"time"
)

// ErrorSeverity represents the severity of an error
type ErrorSeverity int

const (
	SeverityFatal   ErrorSeverity = iota // 処理続行不可
	SeverityError                        // エラーだが処理は継続可能
	SeverityWarning                      // 警告レベル
	SeverityInfo                         // 情報レベル
)

// ErrorCategory represents the category of an error
type ErrorCategory string

const (
	CategoryConfig    ErrorCategory = "CONFIG"     // 設定関連
	CategoryParse     ErrorCategory = "PARSE"      // パース関連
	CategoryAnalysis  ErrorCategory = "ANALYSIS"   // 解析関連
	CategoryIO        ErrorCategory = "IO"         // 入出力関連
	CategoryInternal  ErrorCategory = "INTERNAL"   // 内部エラー
)

// AnalysisError represents an analysis error
type AnalysisError struct {
	ID         string                 // 一意のエラーID
	Category   ErrorCategory          // エラーカテゴリ
	Severity   ErrorSeverity          // 深刻度
	Message    string                 // エラーメッセージ
	Details    map[string]interface{} // 詳細情報
	Location   *ErrorLocation         // エラー発生場所
	Timestamp  time.Time              // 発生時刻
	StackTrace string                 // スタックトレース（デバッグモード時）
	Wrapped    error                  // ラップされた元エラー
}

// ErrorLocation represents the location where an error occurred
type ErrorLocation struct {
	File     string // ファイルパス
	Line     int    // 行番号
	Column   int    // 列番号（該当する場合）
	Function string // 関数名
}

// Error implements the error interface
func (e *AnalysisError) Error() string {
	if e.Location != nil {
		return fmt.Sprintf("[%s] %s at %s:%d - %s", 
			e.Category, e.ID, e.Location.File, e.Location.Line, e.Message)
	}
	return fmt.Sprintf("[%s] %s - %s", e.Category, e.ID, e.Message)
}

// Unwrap returns the wrapped error
func (e *AnalysisError) Unwrap() error {
	return e.Wrapped
}

// NewError creates a new analysis error
func NewError(category ErrorCategory, severity ErrorSeverity, message string) *AnalysisError {
	pc, file, line, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)
	
	return &AnalysisError{
		ID:        generateErrorID(),
		Category:  category,
		Severity:  severity,
		Message:   message,
		Timestamp: time.Now(),
		Location: &ErrorLocation{
			File:     file,
			Line:     line,
			Function: fn.Name(),
		},
		Details: make(map[string]interface{}),
	}
}

// Wrap wraps an error with additional context
func Wrap(err error, message string) *AnalysisError {
	if err == nil {
		return nil
	}
	
	// 既存のAnalysisErrorの場合は情報を保持
	if ae, ok := err.(*AnalysisError); ok {
		ae.Message = fmt.Sprintf("%s: %s", message, ae.Message)
		return ae
	}
	
	// 新規エラーとしてラップ
	newErr := NewError(CategoryInternal, SeverityError, message)
	newErr.Wrapped = err
	return newErr
}

// generateErrorID generates a unique error ID
func generateErrorID() string {
	// 簡単な実装（実際にはより複雑な方法を使用）
	return fmt.Sprintf("ERR_%d", time.Now().UnixNano())
}

// String returns the string representation of severity
func (s ErrorSeverity) String() string {
	switch s {
	case SeverityFatal:
		return "FATAL"
	case SeverityError:
		return "ERROR"
	case SeverityWarning:
		return "WARNING"
	case SeverityInfo:
		return "INFO"
	default:
		return "UNKNOWN"
	}
}