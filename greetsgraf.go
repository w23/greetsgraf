package main

import (
	"flag"
	"log"
)

type Args struct {
	create       bool
	pouet_prods  string
	pouet_groups string
	serve        bool
	listen       string
	usage        bool
	static       string
	index        bool
	pouet_db     string
	greets_db    string
}

func parseArgs() (args Args) {
	flag.BoolVar(&args.create, "create", false, "Create a new database from pouet dumps")
	flag.StringVar(&args.pouet_prods, "prods", "", "pouetdatadump-prods .json.gz file taken from https://data.pouet.net/")
	flag.StringVar(&args.pouet_groups, "groups", "", "pouetdatadump-groups .json.gz file taken from https://data.pouet.net/")
	flag.BoolVar(&args.serve, "serve", false, "Start a server to serve REST API calls")
	flag.StringVar(&args.listen, "listen", "localhost:8000", "Address to listen to and to serve api calls from")
	flag.StringVar(&args.static, "static", "", "(intendede for local debug only) Also serve static data at this path")
	flag.BoolVar(&args.index, "index", false, "Build FTS5 index")
	flag.BoolVar(&args.usage, "help", false, "Print usage")
	flag.StringVar(&args.pouet_db, "pouet-db", "pouet.db", "Readonly database for pouet data")
	flag.StringVar(&args.greets_db, "greets-db", "greets.db", "Writable database for greets data")
	flag.Parse()
	return
}

func main() {
	args := parseArgs()

	if args.usage {
		flag.Usage()
		return
	}

	pouetDB, err := PouetOpen(args.pouet_db)
	if err != nil {
		flag.Usage()
		log.Fatalf("Cannot open pouet database file %s: %v", args.pouet_db, err)
	}

	greetsDB, err := GreetsOpen(args.greets_db)
	if err != nil {
		flag.Usage()
		log.Fatalf("Cannot open greets database file %s: %v", args.greets_db, err)
	}

	if args.create {
		pouetDB.ImportPouet(args.pouet_prods, args.pouet_groups)
		pouetDB.BuildIndex()

		// Migrate greets DB
		greetsDB.AutoMigrate()
	} else if args.index {
		pouetDB.BuildIndex()
	}

	if args.serve {
		listen(&pouetDB, &greetsDB, args.listen, args.static)
	}
}
