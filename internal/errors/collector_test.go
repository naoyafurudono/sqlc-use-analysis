package errors

import (
	"testing"
)

func TestErrorCollector_Add(t *testing.T) {
	collector := NewErrorCollector(10, false)
	
	// 警告の追加
	warning := NewError(CategoryAnalysis, SeverityWarning, "test warning")
	err := collector.Add(warning)
	if err != nil {
		t.Errorf("Add() error = %v", err)
	}
	
	// エラーの追加
	error := NewError(CategoryParse, SeverityError, "test error")
	err = collector.Add(error)
	if err != nil {
		t.Errorf("Add() error = %v", err)
	}
	
	// 結果の確認
	if !collector.HasErrors() {
		t.Error("Expected HasErrors() to be true")
	}
	
	errors := collector.GetErrors()
	if len(errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errors))
	}
	
	warnings := collector.GetWarnings()
	if len(warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(warnings))
	}
}

func TestErrorCollector_MaxErrors(t *testing.T) {
	collector := NewErrorCollector(2, false)
	
	// 最大数までエラーを追加
	for i := 0; i < 2; i++ {
		err := NewError(CategoryAnalysis, SeverityError, "test error")
		if addErr := collector.Add(err); addErr != nil {
			t.Errorf("Add() error = %v", addErr)
		}
	}
	
	// 最大数に達しているので、次のエラーは追加できるはず（まだ境界値）
	// 実際は maxErrors に達した時点でエラーを返すので、実装を確認
	
	// 最大数を超えるエラーを追加
	err := NewError(CategoryAnalysis, SeverityError, "too many errors")
	addErr := collector.Add(err)
	if addErr == nil {
		t.Error("Expected error when adding too many errors")
	}
}

func TestErrorCollector_StopOnFatal(t *testing.T) {
	collector := NewErrorCollector(10, true)
	
	// 致命的エラーの追加
	fatal := NewError(CategoryConfig, SeverityFatal, "fatal error")
	err := collector.Add(fatal)
	if err == nil {
		t.Error("Expected error when adding fatal error with stopOnFatal=true")
	}
}

func TestErrorCollector_GetReport(t *testing.T) {
	collector := NewErrorCollector(10, false)
	
	// 複数のエラーと警告を追加
	collector.Add(NewError(CategoryConfig, SeverityError, "config error"))
	collector.Add(NewError(CategoryParse, SeverityError, "parse error"))
	collector.Add(NewError(CategoryAnalysis, SeverityWarning, "analysis warning"))
	
	report := collector.GetReport()
	
	if report.Summary.TotalErrors != 2 {
		t.Errorf("Expected 2 errors, got %d", report.Summary.TotalErrors)
	}
	
	if report.Summary.TotalWarnings != 1 {
		t.Errorf("Expected 1 warning, got %d", report.Summary.TotalWarnings)
	}
	
	// カテゴリ別の集計確認
	if report.Summary.ByCategory[CategoryConfig] != 1 {
		t.Errorf("Expected 1 config error, got %d", report.Summary.ByCategory[CategoryConfig])
	}
	
	if report.Summary.ByCategory[CategoryParse] != 1 {
		t.Errorf("Expected 1 parse error, got %d", report.Summary.ByCategory[CategoryParse])
	}
	
	if report.Summary.ByCategory[CategoryAnalysis] != 1 {
		t.Errorf("Expected 1 analysis warning, got %d", report.Summary.ByCategory[CategoryAnalysis])
	}
}