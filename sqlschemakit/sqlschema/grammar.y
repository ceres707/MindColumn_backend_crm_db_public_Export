%{
package sqlschema
%}

%union {
	str     string
	boolean bool
	stmt    *CreateTableStmt
	coldef  *ColumnDef
	typ     *TypeName
	sizes   []int
	ccon    ColumnConstraint
	ccons   []ColumnConstraint
	tcon    TableConstraint
	idxcol  IndexedColumn
	idxcols []IndexedColumn
	strs    []string
	expr    Expr
	exprs   []Expr
	fk      *ForeignKeyClause
	fkacts  *fkActions
	defs    *tableDefs
}

%token <str> IDENT STRING NUMBER

%token CREATE TABLE TEMP TEMPORARY IF NOT EXISTS PRIMARY KEY UNIQUE CHECK DEFAULT COLLATE
%token REFERENCES ON DELETE UPDATE CASCADE RESTRICT SET NULLTOK NO ACTION AUTOINCREMENT
%token ASC DESC WITHOUT ROWID CONSTRAINT FOREIGN GENERATED ALWAYS AS STORED VIRTUAL
%token AND OR IS LIKE IN CONFLICT ROLLBACK ABORT FAIL IGNORE REPLACE
%token TRUE FALSE CURRENT_TIME CURRENT_DATE CURRENT_TIMESTAMP
%token LPAREN RPAREN COMMA DOT SEMI
%token EQ NE LT LE GT GE PLUS MINUS STAR SLASH

%left OR
%left AND
%right NOT
%nonassoc EQ NE LT LE GT GE LIKE IS IN
%left PLUS MINUS
%left STAR SLASH
%right UMINUS

%type <stmt> create_table_stmt
%type <boolean> temp_opt if_not_exists_opt without_rowid_opt autoincrement_opt
%type <str> table_name order_opt conflict_clause_opt fk_reaction generated_kind_opt constraint_name_opt type_name_words
%type <defs> table_def_list
%type <coldef> column_def
%type <typ> type_name type_name_opt
%type <sizes> type_size_opt
%type <ccon> column_constraint
%type <ccons> column_constraint_list column_constraint_list_opt
%type <tcon> table_constraint
%type <idxcol> indexed_column
%type <idxcols> indexed_column_list
%type <strs> column_list fk_columns_opt
%type <expr> expr default_value literal
%type <exprs> expr_list arg_list_opt
%type <fk> foreign_key_clause
%type <fkacts> fk_actions_opt

%start create_table_stmt

%%

create_table_stmt:
	CREATE temp_opt TABLE if_not_exists_opt table_name LPAREN table_def_list RPAREN without_rowid_opt
	{
		schema, name := splitTableName($5)
		$$ = &CreateTableStmt{
			Temp:         $2,
			IfNotExists:  $4,
			Schema:       schema,
			Name:         name,
			Columns:      $7.Columns,
			Constraints:  $7.Constraints,
			WithoutRowID: $9,
		}
		yylex.(*Lexer).Result = $$
	}
	;

temp_opt:
	  /* empty */ { $$ = false }
	| TEMP        { $$ = true }
	| TEMPORARY   { $$ = true }
	;

if_not_exists_opt:
	  /* empty */    { $$ = false }
	| IF NOT EXISTS  { $$ = true }
	;

table_name:
	  IDENT              { $$ = $1 }
	| IDENT DOT IDENT    { $$ = $1 + "." + $3 }
	;

without_rowid_opt:
	  /* empty */   { $$ = false }
	| WITHOUT ROWID { $$ = true }
	;

table_def_list:
	  column_def
		{ $$ = &tableDefs{Columns: []*ColumnDef{$1}} }
	| table_constraint
		{ $$ = &tableDefs{Constraints: []TableConstraint{$1}} }
	| table_def_list COMMA column_def
		{ $1.Columns = append($1.Columns, $3); $$ = $1 }
	| table_def_list COMMA table_constraint
		{ $1.Constraints = append($1.Constraints, $3); $$ = $1 }
	;

column_def:
	IDENT type_name_opt column_constraint_list_opt
	{ $$ = &ColumnDef{Name: $1, Type: $2, Constraints: $3} }
	;

type_name_opt:
	  /* empty */ { $$ = nil }
	| type_name   { $$ = $1 }
	;

type_name:
	type_name_words type_size_opt
	{ $$ = &TypeName{Name: $1, Size: $2} }
	;

type_name_words:
	  IDENT                  { $$ = $1 }
	| type_name_words IDENT  { $$ = $1 + " " + $2 }
	;

type_size_opt:
	  /* empty */                        { $$ = nil }
	| LPAREN NUMBER RPAREN               { $$ = []int{atoiSafe($2)} }
	| LPAREN NUMBER COMMA NUMBER RPAREN  { $$ = []int{atoiSafe($2), atoiSafe($4)} }
	;

column_constraint_list_opt:
	  /* empty */             { $$ = nil }
	| column_constraint_list  { $$ = $1 }
	;

column_constraint_list:
	  column_constraint                         { $$ = []ColumnConstraint{$1} }
	| column_constraint_list column_constraint  { $$ = append($1, $2) }
	;

column_constraint:
	  PRIMARY KEY order_opt autoincrement_opt conflict_clause_opt
		{ $$ = &PrimaryKeyConstraint{Order: $3, AutoIncrement: $4, Conflict: $5} }
	| NOT NULLTOK conflict_clause_opt
		{ $$ = &NotNullConstraint{Conflict: $3} }
	| NULLTOK
		{ $$ = &NullConstraint{} }
	| UNIQUE conflict_clause_opt
		{ $$ = &UniqueConstraint{Conflict: $2} }
	| CHECK LPAREN expr RPAREN
		{ $$ = &CheckConstraint{Expr: $3} }
	| DEFAULT default_value
		{ $$ = &DefaultConstraint{Value: $2} }
	| COLLATE IDENT
		{ $$ = &CollateConstraint{Collation: $2} }
	| foreign_key_clause
		{ $$ = &ForeignKeyColumnConstraint{Clause: $1} }
	| GENERATED ALWAYS AS LPAREN expr RPAREN generated_kind_opt
		{ $$ = &GeneratedConstraint{Expr: $5, Kind: $7} }
	| AS LPAREN expr RPAREN generated_kind_opt
		{ $$ = &GeneratedConstraint{Expr: $3, Kind: $5} }
	;

default_value:
	  literal            { $$ = $1 }
	| LPAREN expr RPAREN { $$ = &ParenExpr{X: $2} }
	| MINUS NUMBER       { $$ = &UnaryExpr{Op: "-", X: &Literal{Kind: "number", Value: $2}} }
	| PLUS NUMBER        { $$ = &Literal{Kind: "number", Value: $2} }
	;

order_opt:
	  /* empty */ { $$ = "" }
	| ASC         { $$ = "ASC" }
	| DESC        { $$ = "DESC" }
	;

autoincrement_opt:
	  /* empty */   { $$ = false }
	| AUTOINCREMENT { $$ = true }
	;

conflict_clause_opt:
	  /* empty */         { $$ = "" }
	| ON CONFLICT ROLLBACK { $$ = "ROLLBACK" }
	| ON CONFLICT ABORT    { $$ = "ABORT" }
	| ON CONFLICT FAIL     { $$ = "FAIL" }
	| ON CONFLICT IGNORE   { $$ = "IGNORE" }
	| ON CONFLICT REPLACE  { $$ = "REPLACE" }
	;

foreign_key_clause:
	REFERENCES IDENT fk_columns_opt fk_actions_opt
	{ $$ = &ForeignKeyClause{Table: $2, Columns: $3, OnDelete: $4.onDelete, OnUpdate: $4.onUpdate} }
	;

fk_columns_opt:
	  /* empty */               { $$ = nil }
	| LPAREN column_list RPAREN { $$ = $2 }
	;

fk_actions_opt:
	  /* empty */                         { $$ = &fkActions{} }
	| fk_actions_opt ON DELETE fk_reaction { $1.onDelete = $4; $$ = $1 }
	| fk_actions_opt ON UPDATE fk_reaction { $1.onUpdate = $4; $$ = $1 }
	;

fk_reaction:
	  CASCADE     { $$ = "CASCADE" }
	| RESTRICT    { $$ = "RESTRICT" }
	| SET NULLTOK { $$ = "SET NULL" }
	| SET DEFAULT { $$ = "SET DEFAULT" }
	| NO ACTION   { $$ = "NO ACTION" }
	;

generated_kind_opt:
	  /* empty */ { $$ = "" }
	| STORED      { $$ = "STORED" }
	| VIRTUAL     { $$ = "VIRTUAL" }
	;

table_constraint:
	  constraint_name_opt PRIMARY KEY LPAREN indexed_column_list RPAREN conflict_clause_opt
		{ $$ = &TablePrimaryKey{Name: $1, Columns: $5, Conflict: $7} }
	| constraint_name_opt UNIQUE LPAREN indexed_column_list RPAREN conflict_clause_opt
		{ $$ = &TableUnique{Name: $1, Columns: $4, Conflict: $6} }
	| constraint_name_opt CHECK LPAREN expr RPAREN
		{ $$ = &TableCheck{Name: $1, Expr: $4} }
	| constraint_name_opt FOREIGN KEY LPAREN column_list RPAREN foreign_key_clause
		{ $$ = &TableForeignKey{Name: $1, Columns: $5, Ref: *$7} }
	;

constraint_name_opt:
	  /* empty */      { $$ = "" }
	| CONSTRAINT IDENT { $$ = $2 }
	;

indexed_column_list:
	  indexed_column                             { $$ = []IndexedColumn{$1} }
	| indexed_column_list COMMA indexed_column   { $$ = append($1, $3) }
	;

indexed_column:
	IDENT order_opt { $$ = IndexedColumn{Name: $1, Order: $2} }
	;

column_list:
	  IDENT                   { $$ = []string{$1} }
	| column_list COMMA IDENT { $$ = append($1, $3) }
	;

expr:
	  expr OR expr                        { $$ = &BinaryExpr{Op: "OR", Left: $1, Right: $3} }
	| expr AND expr                       { $$ = &BinaryExpr{Op: "AND", Left: $1, Right: $3} }
	| NOT expr                            { $$ = &UnaryExpr{Op: "NOT", X: $2} }
	| expr EQ expr                        { $$ = &BinaryExpr{Op: "=", Left: $1, Right: $3} }
	| expr NE expr                        { $$ = &BinaryExpr{Op: "!=", Left: $1, Right: $3} }
	| expr LT expr                        { $$ = &BinaryExpr{Op: "<", Left: $1, Right: $3} }
	| expr LE expr                        { $$ = &BinaryExpr{Op: "<=", Left: $1, Right: $3} }
	| expr GT expr                        { $$ = &BinaryExpr{Op: ">", Left: $1, Right: $3} }
	| expr GE expr                        { $$ = &BinaryExpr{Op: ">=", Left: $1, Right: $3} }
	| expr LIKE expr                      { $$ = &BinaryExpr{Op: "LIKE", Left: $1, Right: $3} }
	| expr IS expr                        { $$ = &BinaryExpr{Op: "IS", Left: $1, Right: $3} }
	| expr IN LPAREN expr_list RPAREN     { $$ = &InExpr{X: $1, List: $4} }
	| expr NOT IN LPAREN expr_list RPAREN { $$ = &UnaryExpr{Op: "NOT", X: &InExpr{X: $1, List: $5}} }
	| expr PLUS expr                      { $$ = &BinaryExpr{Op: "+", Left: $1, Right: $3} }
	| expr MINUS expr                     { $$ = &BinaryExpr{Op: "-", Left: $1, Right: $3} }
	| expr STAR expr                      { $$ = &BinaryExpr{Op: "*", Left: $1, Right: $3} }
	| expr SLASH expr                     { $$ = &BinaryExpr{Op: "/", Left: $1, Right: $3} }
	| MINUS expr %prec UMINUS             { $$ = &UnaryExpr{Op: "-", X: $2} }
	| LPAREN expr RPAREN                  { $$ = &ParenExpr{X: $2} }
	| IDENT LPAREN arg_list_opt RPAREN    { $$ = &FuncCall{Name: $1, Args: $3} }
	| IDENT                               { $$ = &Ident{Name: $1} }
	| literal                             { $$ = $1 }
	;

arg_list_opt:
	  /* empty */ { $$ = nil }
	| expr_list   { $$ = $1 }
	| STAR        { $$ = []Expr{&Ident{Name: "*"}} }
	;

expr_list:
	  expr                  { $$ = []Expr{$1} }
	| expr_list COMMA expr  { $$ = append($1, $3) }
	;

literal:
	  STRING            { $$ = &Literal{Kind: "string", Value: $1} }
	| NUMBER            { $$ = &Literal{Kind: "number", Value: $1} }
	| NULLTOK           { $$ = &Literal{Kind: "null"} }
	| TRUE              { $$ = &Literal{Kind: "bool", Value: "true"} }
	| FALSE             { $$ = &Literal{Kind: "bool", Value: "false"} }
	| CURRENT_TIME      { $$ = &Literal{Kind: "keyword", Value: "CURRENT_TIME"} }
	| CURRENT_DATE      { $$ = &Literal{Kind: "keyword", Value: "CURRENT_DATE"} }
	| CURRENT_TIMESTAMP { $$ = &Literal{Kind: "keyword", Value: "CURRENT_TIMESTAMP"} }
	;

%%