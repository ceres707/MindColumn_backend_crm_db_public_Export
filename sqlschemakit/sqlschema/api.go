package sqlschema

//go:generate goyacc -o grammar.go grammar.y

import "fmt"

// ParseCreateTable parses a single SQLite CREATE TABLE statement and
// returns its AST. A trailing semicolon is optional.
func ParseCreateTable(sql string) (*CreateTableStmt, error) {
	lexer := NewLexer(sql)
	if yyParse(lexer) != 0 {
		if lexer.err != nil {
			return nil, lexer.err
		}
		return nil, fmt.Errorf("parse error")
	}
	if lexer.Result == nil {
		return nil, fmt.Errorf("no CREATE TABLE statement found")
	}
	return lexer.Result, nil
}