package main

import (
	"flag"
	"log"
)

type Args struct {
	db           string
	create       bool
	pouet_prods  string
	pouet_groups string
	serve        bool
	listen       string
	usage        bool
	static       string
	index        bool
}

func parseArgs() (args Args) {
	flag.StringVar(&args.db, "db", "greets.db", "Sqlite3 database filename")
	flag.BoolVar(&args.create, "create", false, "Create a new database from pouet dumps")
	flag.StringVar(&args.pouet_prods, "prods", "", "pouetdatadump-prods .json.gz file taken from https://data.pouet.net/")
	flag.StringVar(&args.pouet_groups, "groups", "", "pouetdatadump-groups .json.gz file taken from https://data.pouet.net/")
	flag.BoolVar(&args.serve, "serve", false, "Start a server to serve REST API calls")
	flag.StringVar(&args.listen, "listen", "localhost:8000", "Address to listen to and to serve api calls from")
	flag.StringVar(&args.static, "static", "", "(intendede for local debug only) Also serve static data at this path")
	flag.BoolVar(&args.index, "index", false, "Build FTS5 index")
	flag.BoolVar(&args.usage, "help", false, "Print usage")
	flag.Parse()
	return
}

func main() {
	args := parseArgs()

	if args.usage {
		flag.Usage()
		return
	}

	db, err := DatabaseOpen(args.db)
	if err != nil {
		flag.Usage()
		log.Fatalf("Cannot open database file %s: %v", args.db, err)
	}

	if args.create {
		db.ImportPouet(args.pouet_prods, args.pouet_groups)
		db.BuildIndex()
	} else if args.index {
		db.BuildIndex()
	}

	if args.serve {
		listen(db, args.listen, args.static)
	}
}
