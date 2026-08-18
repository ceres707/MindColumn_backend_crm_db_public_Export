package sqlemit

import "sqlschemakit/sqlschema"

// Dialect converts a parsed CREATE TABLE AST into another SQL dialect's DDL.
// Register implementations via init() in dialect-specific files so adding
// a new dialect never requires touching this file.
type Dialect interface {
	Name() string
	Emit(stmt *sqlschema.CreateTableStmt) (string, error)
}

var registry = map[string]Dialect{}

func Register(d Dialect)         { registry[d.Name()] = d }
func Get(name string) (Dialect, bool) { d, ok := registry[name]; return d, ok }

func Names() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	return names
}