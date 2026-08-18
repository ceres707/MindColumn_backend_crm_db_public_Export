package sqllint

import (
	"fmt"
	"sort"
	"strings"

	"sqlschemakit/sqlschema"
)

type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarning
	SeverityError
)

func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	default:
		return "unknown"
	}
}

type Finding struct {
	Tag      string
	Severity Severity
	Column   string // empty for table-level findings
	Message  string
}

func (f Finding) String() string {
	if f.Column != "" {
		return fmt.Sprintf("[%s] %s (%s): %s", strings.ToUpper(f.Severity.String()), f.Tag, f.Column, f.Message)
	}
	return fmt.Sprintf("[%s] %s: %s", strings.ToUpper(f.Severity.String()), f.Tag, f.Message)
}

type checkFunc func(*sqlschema.CreateTableStmt) []Finding

var registry = map[string]checkFunc{
	"pk":              checkPrimaryKey,
	"fk-index":        checkForeignKeyIndex,
	"type-affinity":   checkTypeAffinity,
	"nullable-unique": checkNullableUnique,
}

// Tags returns all registered check tags sorted — useful for -list-tags.
func Tags() []string {
	tags := make([]string, 0, len(registry))
	for t := range registry {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return tags
}

// Run executes the named checks (or all checks if tags is empty) against stmt.
func Run(stmt *sqlschema.CreateTableStmt, tags ...string) []Finding {
	active := registry
	if len(tags) > 0 {
		active = make(map[string]checkFunc, len(tags))
		for _, t := range tags {
			if fn, ok := registry[t]; ok {
				active[t] = fn
			}
		}
	}
	var findings []Finding
	for _, fn := range active {
		findings = append(findings, fn(stmt)...)
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Tag != findings[j].Tag {
			return findings[i].Tag < findings[j].Tag
		}
		return findings[i].Column < findings[j].Column
	})
	return findings
}

// checkPrimaryKey flags tables with no explicit PRIMARY KEY.
func checkPrimaryKey(stmt *sqlschema.CreateTableStmt) []Finding {
	for _, col := range stmt.Columns {
		for _, c := range col.Constraints {
			if _, ok := c.(*sqlschema.PrimaryKeyConstraint); ok {
				return nil
			}
		}
	}
	for _, tc := range stmt.Constraints {
		if _, ok := tc.(*sqlschema.TablePrimaryKey); ok {
			return nil
		}
	}
	return []Finding{{
		Tag:      "pk",
		Severity: SeverityWarning,
		Message:  fmt.Sprintf("table %q has no explicit PRIMARY KEY; falls back to SQLite implicit rowid", stmt.Name),
	}}
}

// checkForeignKeyIndex reminds that SQLite never auto-indexes FK columns.
func checkForeignKeyIndex(stmt *sqlschema.CreateTableStmt) []Finding {
	var findings []Finding
	for _, col := range stmt.Columns {
		for _, c := range col.Constraints {
			if fkc, ok := c.(*sqlschema.ForeignKeyColumnConstraint); ok {
				findings = append(findings, Finding{
					Tag:      "fk-index",
					Severity: SeverityInfo,
					Column:   col.Name,
					Message:  fmt.Sprintf("references %q; SQLite does not auto-index FK columns — consider CREATE INDEX", fkc.Clause.Table),
				})
			}
		}
	}
	for _, tc := range stmt.Constraints {
		if fk, ok := tc.(*sqlschema.TableForeignKey); ok {
			findings = append(findings, Finding{
				Tag:      "fk-index",
				Severity: SeverityInfo,
				Message:  fmt.Sprintf("columns %v reference %q; consider adding an index", fk.Columns, fk.Ref.Table),
			})
		}
	}
	return findings
}

// checkTypeAffinity flags declared types that don't map cleanly to a
// SQLite affinity — they still work but are often unintentional.
func checkTypeAffinity(stmt *sqlschema.CreateTableStmt) []Finding {
	var findings []Finding
	for _, col := range stmt.Columns {
		if col.Type == nil {
			continue
		}
		u := strings.ToUpper(col.Type.Name)
		if sqliteAffinity(u) == "NUMERIC" && u != "NUMERIC" && !strings.Contains(u, "DECIMAL") {
			findings = append(findings, Finding{
				Tag:      "type-affinity",
				Severity: SeverityInfo,
				Column:   col.Name,
				Message:  fmt.Sprintf("type %q resolves to NUMERIC affinity; prefer INT, TEXT, BLOB, REAL, or DECIMAL to be explicit", col.Type.Name),
			})
		}
	}
	return findings
}

func sqliteAffinity(u string) string {
	switch {
	case strings.Contains(u, "INT"):
		return "INTEGER"
	case strings.Contains(u, "CHAR"), strings.Contains(u, "CLOB"), strings.Contains(u, "TEXT"):
		return "TEXT"
	case strings.Contains(u, "BLOB"), u == "":
		return "BLOB"
	case strings.Contains(u, "REAL"), strings.Contains(u, "FLOA"), strings.Contains(u, "DOUB"):
		return "REAL"
	default:
		return "NUMERIC"
	}
}

// checkNullableUnique flags UNIQUE columns without NOT NULL: SQLite treats
// NULLs as pairwise distinct, allowing many NULL rows silently.
func checkNullableUnique(stmt *sqlschema.CreateTableStmt) []Finding {
	var findings []Finding
	for _, col := range stmt.Columns {
		hasUnique, hasNotNull := false, false
		for _, c := range col.Constraints {
			switch c.(type) {
			case *sqlschema.UniqueConstraint:
				hasUnique = true
			case *sqlschema.NotNullConstraint:
				hasNotNull = true
			case *sqlschema.PrimaryKeyConstraint:
				hasNotNull = true
			}
		}
		if hasUnique && !hasNotNull {
			findings = append(findings, Finding{
				Tag:      "nullable-unique",
				Severity: SeverityInfo,
				Column:   col.Name,
				Message:  "UNIQUE but nullable; multiple NULL rows are allowed since NULLs are never equal in SQLite",
			})
		}
	}
	return findings
}