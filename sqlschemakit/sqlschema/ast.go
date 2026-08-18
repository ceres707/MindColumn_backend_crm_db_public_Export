package sqlschema

// CreateTableStmt is the root AST node produced by ParseCreateTable.
type CreateTableStmt struct {
	Temp         bool
	IfNotExists  bool
	Schema       string // empty if unqualified
	Name         string
	Columns      []*ColumnDef
	Constraints  []TableConstraint
	WithoutRowID bool
}

type ColumnDef struct {
	Name        string
	Type        *TypeName // nil if column has no declared type (legal in SQLite)
	Constraints []ColumnConstraint
}

type TypeName struct {
	Name string // e.g. "VARCHAR", "DOUBLE PRECISION"
	Size []int  // 0, 1 or 2 entries: (N) or (N,M)
}

// --- Column constraints ---

type ColumnConstraint interface{ isColumnConstraint() }

type PrimaryKeyConstraint struct {
	Order         string // "", "ASC", "DESC"
	AutoIncrement bool
	Conflict      string
}
type NotNullConstraint struct{ Conflict string }
type NullConstraint struct{}
type UniqueConstraint struct{ Conflict string }
type CheckConstraint struct{ Expr Expr }
type DefaultConstraint struct{ Value Expr }
type CollateConstraint struct{ Collation string }
type ForeignKeyColumnConstraint struct{ Clause *ForeignKeyClause }
type GeneratedConstraint struct {
	Expr Expr
	Kind string // "", "STORED", "VIRTUAL"
}

func (*PrimaryKeyConstraint) isColumnConstraint()       {}
func (*NotNullConstraint) isColumnConstraint()          {}
func (*NullConstraint) isColumnConstraint()             {}
func (*UniqueConstraint) isColumnConstraint()           {}
func (*CheckConstraint) isColumnConstraint()            {}
func (*DefaultConstraint) isColumnConstraint()          {}
func (*CollateConstraint) isColumnConstraint()          {}
func (*ForeignKeyColumnConstraint) isColumnConstraint() {}
func (*GeneratedConstraint) isColumnConstraint()        {}

// --- Table-level constraints ---

type TableConstraint interface{ isTableConstraint() }

type TablePrimaryKey struct {
	Name     string
	Columns  []IndexedColumn
	Conflict string
}
type TableUnique struct {
	Name     string
	Columns  []IndexedColumn
	Conflict string
}
type TableCheck struct {
	Name string
	Expr Expr
}
type TableForeignKey struct {
	Name    string
	Columns []string
	Ref     ForeignKeyClause
}

func (*TablePrimaryKey) isTableConstraint() {}
func (*TableUnique) isTableConstraint()     {}
func (*TableCheck) isTableConstraint()      {}
func (*TableForeignKey) isTableConstraint() {}

type IndexedColumn struct {
	Name  string
	Order string // "", "ASC", "DESC"
}

type ForeignKeyClause struct {
	Table    string
	Columns  []string // referenced columns; may be empty (infers PK)
	OnDelete string
	OnUpdate string
}

// --- Expressions (used inside CHECK / DEFAULT / GENERATED) ---

type Expr interface{ isExpr() }

type Ident struct{ Name string }
type Literal struct {
	Kind  string // "string" | "number" | "null" | "bool" | "keyword"
	Value string
}
type BinaryExpr struct {
	Op          string
	Left, Right Expr
}
type UnaryExpr struct {
	Op string
	X  Expr
}
type FuncCall struct {
	Name string
	Args []Expr
}
type ParenExpr struct{ X Expr }
type InExpr struct {
	X    Expr
	List []Expr
}

func (*Ident) isExpr()      {}
func (*Literal) isExpr()    {}
func (*BinaryExpr) isExpr() {}
func (*UnaryExpr) isExpr()  {}
func (*FuncCall) isExpr()   {}
func (*ParenExpr) isExpr()  {}
func (*InExpr) isExpr()     {}

// --- Internal accumulator types used only during parsing ---

type tableDefs struct {
	Columns     []*ColumnDef
	Constraints []TableConstraint
}

type fkActions struct {
	onDelete string
	onUpdate string
}

func splitTableName(s string) (schema, name string) {
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			return s[:i], s[i+1:]
		}
	}
	return "", s
}

func atoiSafe(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}