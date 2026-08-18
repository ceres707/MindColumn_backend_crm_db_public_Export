package sqlschema

import (
	"fmt"
	"strings"
	"unicode"
)

var keywords = map[string]int{
	"CREATE": CREATE, "TABLE": TABLE, "TEMP": TEMP, "TEMPORARY": TEMPORARY,
	"IF": IF, "NOT": NOT, "EXISTS": EXISTS, "PRIMARY": PRIMARY, "KEY": KEY,
	"UNIQUE": UNIQUE, "CHECK": CHECK, "DEFAULT": DEFAULT, "COLLATE": COLLATE,
	"REFERENCES": REFERENCES, "ON": ON, "DELETE": DELETE, "UPDATE": UPDATE,
	"CASCADE": CASCADE, "RESTRICT": RESTRICT, "SET": SET, "NULL": NULLTOK,
	"NO": NO, "ACTION": ACTION, "AUTOINCREMENT": AUTOINCREMENT,
	"ASC": ASC, "DESC": DESC, "WITHOUT": WITHOUT, "ROWID": ROWID,
	"CONSTRAINT": CONSTRAINT, "FOREIGN": FOREIGN, "GENERATED": GENERATED,
	"ALWAYS": ALWAYS, "AS": AS, "STORED": STORED, "VIRTUAL": VIRTUAL,
	"AND": AND, "OR": OR, "IS": IS, "LIKE": LIKE, "IN": IN,
	"CONFLICT": CONFLICT, "ROLLBACK": ROLLBACK, "ABORT": ABORT, "FAIL": FAIL,
	"IGNORE": IGNORE, "REPLACE": REPLACE, "TRUE": TRUE, "FALSE": FALSE,
	"CURRENT_TIME": CURRENT_TIME, "CURRENT_DATE": CURRENT_DATE,
	"CURRENT_TIMESTAMP": CURRENT_TIMESTAMP,
}

type Lexer struct {
	input  []rune
	pos    int
	line   int
	col    int
	Result *CreateTableStmt
	err    error
}

func NewLexer(input string) *Lexer {
	return &Lexer{input: []rune(input), line: 1, col: 1}
}

func (l *Lexer) Err() error { return l.err }

func (l *Lexer) Error(s string) {
	if l.err == nil {
		l.err = fmt.Errorf("line %d, col %d: %s", l.line, l.col, s)
	}
}

func (l *Lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) peekAt(off int) rune {
	if l.pos+off >= len(l.input) {
		return 0
	}
	return l.input[l.pos+off]
}

func (l *Lexer) advance() rune {
	ch := l.input[l.pos]
	l.pos++
	if ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return ch
}

func isIdentStart(r rune) bool { return unicode.IsLetter(r) || r == '_' }
func isIdentPart(r rune) bool  { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' }
func isDigit(r rune) bool      { return r >= '0' && r <= '9' }

func (l *Lexer) skipSpaceAndComments() {
	for l.pos < len(l.input) {
		ch := l.peek()
		switch {
		case unicode.IsSpace(ch):
			l.advance()
		case ch == '-' && l.peekAt(1) == '-':
			for l.pos < len(l.input) && l.peek() != '\n' {
				l.advance()
			}
		case ch == '/' && l.peekAt(1) == '*':
			l.advance()
			l.advance()
			for l.pos < len(l.input) && !(l.peek() == '*' && l.peekAt(1) == '/') {
				l.advance()
			}
			if l.pos < len(l.input) {
				l.advance()
				l.advance()
			}
		default:
			return
		}
	}
}

// Lex implements the yyLexer interface expected by the generated parser.
func (l *Lexer) Lex(lval *yySymType) int {
	l.skipSpaceAndComments()
	if l.pos >= len(l.input) {
		return 0
	}
	ch := l.peek()
	switch {
	case isDigit(ch):
		return l.lexNumber(lval)
	case ch == '\'':
		return l.lexString(lval)
	case ch == '"' || ch == '`':
		return l.lexQuotedIdent(lval, ch)
	case ch == '[':
		return l.lexBracketIdent(lval)
	case isIdentStart(ch):
		return l.lexIdentOrKeyword(lval)
	default:
		return l.lexPunct(lval)
	}
}

func (l *Lexer) lexIdentOrKeyword(lval *yySymType) int {
	start := l.pos
	for l.pos < len(l.input) && isIdentPart(l.peek()) {
		l.advance()
	}
	word := string(l.input[start:l.pos])
	if tok, ok := keywords[strings.ToUpper(word)]; ok {
		lval.str = word
		return tok
	}
	lval.str = word
	return IDENT
}

func (l *Lexer) lexNumber(lval *yySymType) int {
	start := l.pos
	for l.pos < len(l.input) && isDigit(l.peek()) {
		l.advance()
	}
	if l.peek() == '.' && isDigit(l.peekAt(1)) {
		l.advance()
		for l.pos < len(l.input) && isDigit(l.peek()) {
			l.advance()
		}
	}
	if l.peek() == 'e' || l.peek() == 'E' {
		l.advance()
		if l.peek() == '+' || l.peek() == '-' {
			l.advance()
		}
		for l.pos < len(l.input) && isDigit(l.peek()) {
			l.advance()
		}
	}
	lval.str = string(l.input[start:l.pos])
	return NUMBER
}

func (l *Lexer) lexString(lval *yySymType) int {
	l.advance() // opening '
	var b strings.Builder
	for l.pos < len(l.input) {
		ch := l.advance()
		if ch == '\'' {
			if l.peek() == '\'' {
				b.WriteRune('\'')
				l.advance()
				continue
			}
			lval.str = b.String()
			return STRING
		}
		b.WriteRune(ch)
	}
	l.Error("unterminated string literal")
	lval.str = b.String()
	return STRING
}

func (l *Lexer) lexQuotedIdent(lval *yySymType, quote rune) int {
	l.advance()
	var b strings.Builder
	for l.pos < len(l.input) {
		ch := l.advance()
		if ch == quote {
			if l.peek() == quote {
				b.WriteRune(quote)
				l.advance()
				continue
			}
			lval.str = b.String()
			return IDENT
		}
		b.WriteRune(ch)
	}
	l.Error("unterminated quoted identifier")
	lval.str = b.String()
	return IDENT
}

func (l *Lexer) lexBracketIdent(lval *yySymType) int {
	l.advance() // '['
	start := l.pos
	for l.pos < len(l.input) && l.peek() != ']' {
		l.advance()
	}
	lval.str = string(l.input[start:l.pos])
	if l.pos < len(l.input) {
		l.advance() // ']'
	}
	return IDENT
}

func (l *Lexer) lexPunct(lval *yySymType) int {
	ch := l.advance()
	switch ch {
	case '(':
		return LPAREN
	case ')':
		return RPAREN
	case ',':
		return COMMA
	case '.':
		return DOT
	case ';':
		return SEMI
	case '=':
		return EQ
	case '!':
		if l.peek() == '=' {
			l.advance()
			return NE
		}
	case '<':
		if l.peek() == '=' {
			l.advance()
			return LE
		}
		if l.peek() == '>' {
			l.advance()
			return NE
		}
		return LT
	case '>':
		if l.peek() == '=' {
			l.advance()
			return GE
		}
		return GT
	case '+':
		return PLUS
	case '-':
		return MINUS
	case '*':
		return STAR
	case '/':
		return SLASH
	}
	l.Error(fmt.Sprintf("unexpected character %q", ch))
	return int(ch)
}