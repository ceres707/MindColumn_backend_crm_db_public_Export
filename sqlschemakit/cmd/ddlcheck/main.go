package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"sqlschemakit/sqlemit"
	"sqlschemakit/sqllint"
	"sqlschemakit/sqlschema"
)

func main() {
	dialect  := flag.String("dialect", "", "emit target dialect (e.g. duckdb); omit to lint only")
	verbose  := flag.Int("v", 0, "verbosity: 0 = warnings+errors only, 1 = include info")
	tags     := flag.String("tags", "", "comma-separated lint tags; omit = run all")
	listTags := flag.Bool("list-tags", false, "print available lint tags and exit")
	flag.Parse()

	if *listTags {
		fmt.Println(strings.Join(sqllint.Tags(), "\n"))
		return
	}

	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: ddlcheck [flags] <schema.sql>")
		flag.PrintDefaults()
		os.Exit(2)
	}

	data, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}

	stmt, err := sqlschema.ParseCreateTable(string(data))
	if err != nil {
		log.Fatal(err)
	}

	var tagList []string
	if *tags != "" {
		tagList = strings.Split(*tags, ",")
	}
	for _, f := range sqllint.Run(stmt, tagList...) {
		if *verbose > 0 || f.Severity >= sqllint.SeverityWarning {
			fmt.Println(f)
		}
	}

	if *dialect != "" {
		d, ok := sqlemit.Get(*dialect)
		if !ok {
			log.Fatalf("unknown dialect %q — available: %v", *dialect, sqlemit.Names())
		}
		out, err := d.Emit(stmt)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(out)
	}
}