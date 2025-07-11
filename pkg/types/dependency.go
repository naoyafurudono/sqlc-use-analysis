package types

// DependencyPath represents a path from a function to a table
type DependencyPath struct {
	From         string   `json:"from"`          // 起点の関数
	To           string   `json:"to"`            // sqlcメソッド
	Intermediate []string `json:"intermediate"`  // 経由する関数のリスト
	Type         DependencyType `json:"type"`    // 依存関係の種類
}

// DependencyType represents the type of dependency
type DependencyType string

const (
	DirectDependency   DependencyType = "direct"   // 直接sqlcメソッドを呼び出す
	IndirectDependency DependencyType = "indirect" // 他の関数を経由して呼び出す
)

// CallGraph represents a call graph
type CallGraph struct {
	Nodes map[string]*CallNode `json:"nodes"`
	Edges map[string][]*CallEdge `json:"edges"`
}

// CallNode represents a node in the call graph
type CallNode struct {
	FunctionName string          `json:"function_name"`
	IsSQLCMethod bool            `json:"is_sqlc_method"`
	TableOps     []TableOperation `json:"table_ops,omitempty"`
}

// CallEdge represents an edge in the call graph
type CallEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Line int    `json:"line"`
}

// Operation represents a database operation
type Operation string

const (
	OpSelect Operation = "SELECT"
	OpInsert Operation = "INSERT"
	OpUpdate Operation = "UPDATE"
	OpDelete Operation = "DELETE"
)

// String returns the string representation of an operation
func (o Operation) String() string {
	return string(o)
}

// IsValid checks if the operation is valid
func (o Operation) IsValid() bool {
	switch o {
	case OpSelect, OpInsert, OpUpdate, OpDelete:
		return true
	default:
		return false
	}
}