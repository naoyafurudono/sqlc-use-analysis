# SQLC Use Analysis - Design Documentation

このディレクトリには、sqlc-use-analysisプロジェクトの設計ドキュメントが含まれています。

## 📋 目次

### 🎯 **基本仕様・要件**
- **[spec.md](spec.md)** - プロジェクトの基本仕様と要件定義（日本語）
- **[README.md](../README.md)** - プロジェクト概要とクイックスタートガイド

### 🏗️ **システム設計**
- **[design.md](design.md)** - システム設計の中核文書（日本語）
- **[integrated_design.md](integrated_design.md)** - 統合設計書（英語）
- **[architecture_design.md](architecture_design.md)** - システム全体のアーキテクチャ設計
- **[design-improvements.md](design-improvements.md)** - 「A Philosophy of Software Design」に基づく設計改善

### 🔧 **モジュール別設計**
- **[sql_analyzer_design.md](sql_analyzer_design.md)** - SQL解析モジュールの詳細設計
- **[go_analyzer_design.md](go_analyzer_design.md)** - Go静的解析モジュールの設計
- **[dependency_mapper_design.md](dependency_mapper_design.md)** - 依存関係マッピングエンジンの設計
- **[io_interface_design.md](io_interface_design.md)** - 入出力インターフェースの設計
- **[config_management_design.md](config_management_design.md)** - 設定管理システムの設計
- **[error_handling_policy.md](error_handling_policy.md)** - エラーハンドリングの方針と実装

### 📅 **開発計画・管理**
- **[development_plan.md](development_plan.md)** - 13週間の開発計画とフェーズ管理
- **[detailed_milestones.md](detailed_milestones.md)** - 詳細なマイルストーンとタスク分解
- **[development_environment.md](development_environment.md)** - 開発環境の設定とツール構成
- **[risk_analysis.md](risk_analysis.md)** - リスク分析と緩和策

## 🎯 **ドキュメントの使い方**

### 🔰 **初めての方**
1. **[spec.md](spec.md)** - プロジェクトの目的と要件を理解
2. **[design.md](design.md)** - システム設計の基本概念を把握
3. **[development_plan.md](development_plan.md)** - 開発の進捗と計画を確認

### 🏗️ **アーキテクチャを理解したい方**
1. **[architecture_design.md](architecture_design.md)** - システム全体の構造
2. **[integrated_design.md](integrated_design.md)** - 統合設計の詳細
3. **[design-improvements.md](design-improvements.md)** - 最新の設計改善

### 🔧 **実装を理解したい方**
1. **[sql_analyzer_design.md](sql_analyzer_design.md)** - SQL解析の仕組み
2. **[go_analyzer_design.md](go_analyzer_design.md)** - Go解析の仕組み
3. **[dependency_mapper_design.md](dependency_mapper_design.md)** - 依存関係マッピングの仕組み

### 📊 **開発管理を理解したい方**
1. **[development_plan.md](development_plan.md)** - 開発計画の全体像
2. **[detailed_milestones.md](detailed_milestones.md)** - 具体的なタスクと進捗
3. **[risk_analysis.md](risk_analysis.md)** - リスクと対策

## 📈 **開発フェーズ**

### ✅ **完了フェーズ**
- **M1: 基盤構築** - プロジェクト構造、CLI実装、設定管理、エラーハンドリング
- **M2: SQL解析エンジン** - SQLクエリ解析、テーブル抽出、操作種別判定
- **M3: Go静的解析器** - Go AST解析、関数定義抽出、メソッド呼び出し検出
- **M4: 依存関係マッピング** - SQL/Go統合、依存関係構築、出力フォーマット

### 🔄 **進行中・予定フェーズ**
- **M5: エンドツーエンドテスト** - 統合テスト、パフォーマンステスト
- **M6: 本番対応** - エラーハンドリング強化、パフォーマンス最適化
- **M7: リリース準備** - ドキュメント整備、使用例作成

## 🎨 **設計原則**

### 📚 **「A Philosophy of Software Design」の適用**
- **深いモジュール**: 複雑性を隠蔽するシンプルなインターフェース
- **情報隠蔽**: 内部実装の詳細を完全に隠蔽
- **エラーハンドリング**: 構造化されたエラー収集と報告
- **モジュール設計**: 関心の分離と責任の明確化

詳細は **[design-improvements.md](design-improvements.md)** を参照してください。

## 🔍 **品質管理**

### 🧪 **テスト戦略**
- **ユニットテスト**: 90%以上のカバレッジ目標
- **統合テスト**: 実際のプロジェクトでの検証
- **パフォーマンステスト**: 10K+行のコードを5分以内で解析
- **セキュリティテスト**: 脆弱性スキャン

### 📊 **コード品質**
- **リンティング**: golangci-lintによる包括的なルール
- **静的解析**: govulncheckを含む複数のツール
- **CI/CD**: 自動テストとビルド
- **ドキュメント**: 包括的な技術文書

## 🔧 **技術仕様**

### 📥 **入力形式**
- sqlc CodeGeneratorRequest（JSON）
- Go ソースコードパッケージ
- 設定ファイル（sqlc.yaml）

### 📤 **出力形式**
- JSON（メタデータ、function_view、table_view）
- CSV（表形式のエクスポート）
- HTML（視覚的なレポート）

### 🛠️ **主要技術**
- **Go 1.21+**: 主要実装言語
- **go/packages**: Go コード解析
- **go/ast**: 抽象構文木解析
- **sqlc plugin protocol**: 統合方式

## 📞 **サポート**

### 📝 **ドキュメント更新**
ドキュメントの更新や追加が必要な場合は、以下の方針に従ってください：

1. **設計変更**: 対応するdesignドキュメントを更新
2. **新機能**: 関連するモジュール設計文書に追加
3. **API変更**: integrated_design.mdとspec.mdを更新
4. **このREADME**: 新しいドキュメントが追加された場合に更新

### 🔍 **ドキュメント構造**
```
docs/
├── README.md                    # このファイル（目次）
├── spec.md                     # 基本仕様
├── design.md                   # 中核設計
├── integrated_design.md        # 統合設計
├── design-improvements.md      # 設計改善
├── architecture_design.md      # アーキテクチャ
├── sql_analyzer_design.md     # SQL解析設計
├── go_analyzer_design.md      # Go解析設計
├── dependency_mapper_design.md # 依存関係設計
├── io_interface_design.md     # I/O設計
├── config_management_design.md # 設定管理設計
├── error_handling_policy.md   # エラーハンドリング
├── development_plan.md        # 開発計画
├── detailed_milestones.md     # 詳細マイルストーン
├── development_environment.md # 開発環境
└── risk_analysis.md           # リスク分析
```

---

**最終更新**: 2024年7月11日  
**バージョン**: M4フェーズ完了版  
**メンテナー**: Claude Code Assistant