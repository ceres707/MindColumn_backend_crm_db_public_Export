# MindColumn_backend_crm_db_public_Export
Experimental: MindColumn db Imagined forms of transpiling to other database scripts

**Boilerplate made with 2026 best practices, old-tech, new lang**
This is alpha, just started, but _far reaching_
From now on everything is going to be a grammar, DSL first —meaning:

* Input-to-output data verification ➔ grammar
* Apache Kafka messages ➔ grammar
* Language subtypes in text ➔ grammar
* Expression and announcements ➔ grammar
* Using a CRM between SQL dialects ➔ grammar
* Scientific datasets for review ➔ grammar metamodel

### Tips to start with this
* `Fix the conflicts — goyacc -v grammar.y writes y.output; grep it for conflict. The likely spots are the optional trailing clauses (conflict_clause_opt chains) and possibly column_constraint_list greedily consuming — LALR defaults to shift in ambiguous cases, which is almost always what you want here, but read y.output`
* `things like STRICT tables (a real SQLite keyword you'll want to add), multi-column type oddities, or comments inside definitions.`
* Emit a perfect copy of the input schema in SQLite. That's good, isn't it?
* `sqllint.Run findings come back unordered (map iteration order), so if you want stable output for scripting/diffing, sort findings by Tag then Column before printing — trivial to add in main.go`

### SQLite early linter
* `No PRIMARY KEY at all (SQLite gives you an implicit rowid, which is often not what someone intended)`
* `INTEGER PRIMARY KEY without AUTOINCREMENT — fine, but worth flagging since rowid reuse after deletes can surprise people. Declared type names SQLite doesn't recognize under its type affinity rules (e.g. someone writes STRING expecting TEXT affinity — it still "works" via substring matching, but it's a footgun worth surfacing)`
* `FOREIGN KEY with no matching index on the referencing column (SQLite doesn't auto-index FKs, unlike some other engines). Columns marked UNIQUE that are also nullable in ways likely unintended`
* Very early boilerplate: `fk-index check fires on every FK right now since the parser sees one statement at a time and can't cross-reference sibling CREATE INDEX statements. Known false-positive; suppress with -tags=pk,type-affinity,nullable-unique until multi-statement parsing lands.`

### DuckDB early translation
* `The main frictions you'll hit: AUTOINCREMENT has no direct DuckDB equivalent (map to a SEQUENCE or just drop it since DuckDB's rowid semantics differ), WITHOUT ROWID is meaningless in DuckDB, and type affinity is stricter in DuckDB so loose SQLite types need a mapping table.`
* `The DuckDB emitter is a first pass, not exhaustive — it handles the common path (types, PK/NOT NULL/UNIQUE/DEFAULT/CHECK/FK, generated columns) and leaves inline -- note: comments where SQLite semantics have no clean DuckDB equivalent (AUTOINCREMENT, COLLATE, WITHOUT ROWID) rather than silently dropping them. That's a deliberate choice — silent semantic loss during a dialect translation is the kind of bug that's very hard to notice until production.`
* `checkForeignKeyIndex will fire on every single FK right now, since the parser only sees one CREATE TABLE at a time and has no visibility into sibling CREATE INDEX statements in the same file. Once you're parsing whole .sql dumps instead of one statement at a time, that check should cross-reference against parsed indexes to stop being noisy — worth flagging as a known false-positive source for now.`