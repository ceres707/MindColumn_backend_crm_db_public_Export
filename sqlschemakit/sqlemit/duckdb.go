package sqlemit

import (
	"fmt"
	"strings"

	"sqlschemakit/sqlschema"
)

func init() { Register(duckdb{}) }

type duckdb struct{}

func (duckdb) Name() string { return "duckdb" }

var duckdbTypeMap = map[string]string{
	"INT": "INTEGER", "INTEGER": "INTEGER", "TINYINT": "TINYINT",
	"SMALLINT": "SMALLINT", "MEDIUMINT": "INTEGER", "BIGINT": "BIGINT",
	"UNSIGNED BIG INT": "UBIGINT", "INT2": "SMALLINT", "INT8": "BIGINT",
	"CHARACTER": "VARCHAR", "VARCHAR": "VARCHAR", "VARYING CHARACTER": "VARCHAR",
	"NCHAR": "VARCHAR", "NATIVE CHARACTER": "VARCHAR", "NVARCHAR": "VARCHAR",
	"TEXT": "VARCHAR", "CLOB": "VARCHAR",
	"BLOB": "BLOB",
	"REAL": "DOUBLE", "DOUBLE": "DOUBLE", "DOUBLE PRECISION": "DOUBLE", "FLOAT": "FLOAT",
	"NUMERIC": "DECIMAL", "DECIMAL": "DECIMAL",
	"BOOLEAN": "BOOLEAN", "BOOL": "BOOLEAN",
	"DATE": "DATE", "DATETIME": "TIMESTAMP", "TIMESTAMP": "TIMESTAMP",
}

func mapType(t *sqlschema.TypeName) string {
	if t == nil {
		return "VARCHAR"
	}
	u := strings.ToUpper(t.Name)
	mapped, ok := duckdbTypeMap[u]
	if !ok {
		mapped = "VARCHAR"
	}
	if len(t.Size) > 0 && (mapped == "VARCHAR" || mapped == "DECIMAL") {
		parts := make([]string, len(t.Size))
		for i, n := range t.Size {
			parts[i] = fmt.Sprintf("%d", n)
		}
		mapped += "(" + strings.Join(parts, ",") + ")"
	}
	return mapped
}

func (duckdb) Emit(stmt *sqlschema.CreateTableStmt) (string, error) {
	var b strings.Builder
	var notes []string

	b.WriteString("CREATE TABLE ")
	if stmt.IfNotExists {
		b.WriteString("IF NOT EXISTS ")
	}
	if stmt.Schema != "" {
		b.WriteString(stmt.Schema + ".")
	}
	b.WriteString(stmt.Name)
	b.WriteString(" (\n")

	var lines []string
	for _, col := range stmt.Columns {
		line := "    " + col.Name + " " + mapType(col.Type)
		for _, c := range col.Constraints {
			switch cc := c.(type) {
			case *sqlschema.PrimaryKeyConstraint:
				line += " PRIMARY KEY"
				if cc.AutoIncrement {
					notes = append(notes, fmt.Sprintf(
						"-- note: %q used AUTOINCREMENT; DuckDB has no direct equivalent — consider a SEQUENCE", col.Name))
				}
			case *sqlschema.NotNullConstraint:
				line += " NOT NULL"
			case *sqlschema.UniqueConstraint:
				line += " UNIQUE"
			case *sqlschema.DefaultConstraint:
				line += " DEFAULT " + exprToSQL(cc.Value)
			case *sqlschema.CheckConstraint:
				line += " CHECK (" + exprToSQL(cc.Expr) + ")"
			case *sqlschema.CollateConstraint:
				notes = append(notes, fmt.Sprintf(
					"-- note: COLLATE %q on %q dropped; DuckDB collation syntax differs", cc.Collation, col.Name))
			case *sqlschema.ForeignKeyColumnConstraint:
				line += " REFERENCES " + cc.Clause.Table
				if len(cc.Clause.Columns) > 0 {
					line += "(" + strings.Join(cc.Clause.Columns, ", ") + ")"
				}
			case *sqlschema.GeneratedConstraint:
				line += " GENERATED ALWAYS AS (" + exprToSQL(cc.Expr) + ")"
				if cc.Kind != "" {
					line += " " + cc.Kind
				}
			}
		}
		lines = append(lines, line)
	}

	for _, tc := range stmt.Constraints {
		switch c := tc.(type) {
		case *sqlschema.TablePrimaryKey:
			lines = append(lines, "    PRIMARY KEY ("+strings.Join(indexedNames(c.Columns), ", ")+")")
		case *sqlschema.TableUnique:
			lines = append(lines, "    UNIQUE ("+strings.Join(indexedNames(c.Columns), ", ")+")")
		case *sqlschema.TableCheck:
			lines = append(lines, "    CHECK ("+exprToSQL(c.Expr)+")")
		case *sqlschema.TableForeignKey:
			l := "    FOREIGN KEY (" + strings.Join(c.Columns, ", ") + ") REFERENCES " + c.Ref.Table
			if len(c.Ref.Columns) > 0 {
				l += "(" + strings.Join(c.Ref.Columns, ", ") + ")"
			}
			lines = append(lines, l)
		}
	}

	b.WriteString(strings.Join(lines, ",\n"))
	b.WriteString("\n);\n")

	if stmt.WithoutRowID {
		notes = append(notes, "-- note: WITHOUT ROWID dropped; DuckDB has no rowid concept to opt out of")
	}
	for _, n := range notes {
		b.WriteString(n + "\n")
	}
	return b.String(), nil
}

func indexedNames(cols []sqlschema.IndexedColumn) []string {
	names := make([]string, len(cols))
	for i, c := range cols {
		names[i] = c.Name
	}
	return names
}

func exprToSQL(e sqlschema.Expr) string {
	switch x := e.(type) {
	case *sqlschema.Literal:
		switch x.Kind {
		case "string":
			return "'" + strings.ReplaceAll(x.Value, "'", "''") + "'"
		case "null":
			return "NULL"
		default:
			return x.Value
		}
	case *sqlschema.Ident:
		return x.Name
	case *sqlschema.ParenExpr:
		return "(" + exprToSQL(x.X) + ")"
	case *sqlschema.UnaryExpr:
		return x.Op + " " + exprToSQL(x.X)
	case *sqlschema.BinaryExpr:
		return exprToSQL(x.Left) + " " + x.Op + " " + exprToSQL(x.Right)
	case *sqlschema.FuncCall:
		args := make([]string, len(x.Args))
		for i, a := range x.Args {
			args[i] = exprToSQL(a)
		}
		return x.Name + "(" + strings.Join(args, ", ") + ")"
	case *sqlschema.InExpr:
		items := make([]string, len(x.List))
		for i, a := range x.List {
			items[i] = exprToSQL(a)
		}
		return exprToSQL(x.X) + " IN (" + strings.Join(items, ", ") + ")"
	default:
		return ""
	}
}