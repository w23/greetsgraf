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

type SetupArgs struct {
	DBFile      string
	Create      bool
	PouetProds  string
	PouetGroups string
	BuildIndex  bool
}

func SetupDatabase(args SetupArgs) (Database, error) {
	db, err := DatabaseOpen(args.DBFile)
	if err != nil {
		return Database{}, err
	}

	if args.Create {
		db.ImportPouet(args.PouetProds, args.PouetGroups)
		db.BuildIndex()
	} else if args.BuildIndex {
		db.BuildIndex()
	}

	return db, nil
}

func main() {
	args := parseArgs()

	db, err := SetupDatabase(SetupArgs{
		DBFile:      args.db,
		Create:      args.create,
		PouetProds:  args.pouet_prods,
		PouetGroups: args.pouet_groups,
		BuildIndex:  args.index,
	})
	if err != nil {
		flag.Usage()
		log.Fatalf("Cannot setup database %s: %v", args.db, err)
	}

	if args.serve {
		listen(db, args.listen, args.static)
	}
}
