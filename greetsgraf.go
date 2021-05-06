package main

import (
	"os"
	"log"
  "gorm.io/gorm"
  "gorm.io/driver/sqlite"
	"encoding/json"
	"flag"
	"compress/gzip"
	"strconv"
)

type Group struct {
	PouetId int
	Name string
	Disambiguation string
}

type Prod struct {
	PouetId int
	GroupId int
	Name string
}

type Relation struct {
	//gorm.Model
}

func DatabaseOpen(datafile string) (db *gorm.DB, err error) {
	db, err = gorm.Open(sqlite.Open(datafile), &gorm.Config{})
	return
}

func readJsonGz(filename string) (map[string]interface{}, error) {
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("Error opening file %s: %v", filename, err)
		return nil, err
	}

	gz, err := gzip.NewReader(file)
	if err != nil {
		log.Printf("Error unpacking file %s: %v", filename, err)
		return nil, err
	}

	var value map[string]interface{}
	err = json.NewDecoder(gz).Decode(&value)
	if err != nil {
		log.Printf("Error decoding json from file %s: %v", filename, err)
		return nil, err
	}

	return value, err
}

func create(db *gorm.DB, prodsfile string, groupsfile string) {
	if prodsfile == "" || groupsfile == "" {
		flag.Usage()
		log.Fatal("When creating a new db, pouet data dumps are needed\n")
	}

	db.AutoMigrate(&Group{})

	groups, err := readJsonGz(groupsfile)
	if err != nil {
		log.Fatalf("Unable to read groups from file %s: %v", groupsfile, err)
	}

	tx := db.Begin()
	defer tx.Commit()

	groups_array := (groups["groups"]).([]interface{})
	for index, _ := range groups_array {
		group := (groups_array[index]).(map[string]interface{})

		name := group["name"].(string)
		disambiguation := group["disambiguation"].(string)
		pouet_id, err := strconv.ParseInt(group["id"].(string), 10, 64)

		if err != nil {
			log.Printf("wtf id %s", group["id"])
			continue
		}

		dbgroup := Group{
			PouetId: int(pouet_id),
			Name: name,
			Disambiguation: disambiguation,
		}

		log.Printf("%+v", dbgroup)

		tx.Create(&dbgroup)
	}
}

func listen(db *gorm.DB, listen string) {
}

type Args struct {
	db string
	create bool
	pouet_prods string
	pouet_groups string
	serve bool
	listen string
	usage bool
}

func parseArgs() (args Args) {
	flag.StringVar(&args.db, "db", "greets.db", "Sqlite3 database filename")
	flag.BoolVar(&args.create, "create", false, "Create a new database from pouet dumps")
	flag.StringVar(&args.pouet_prods, "prods", "", "pouetdatadump-prods .json.gz file taken from https://data.pouet.net/")
	flag.StringVar(&args.pouet_groups, "groups", "", "pouetdatadump-groups .json.gz file taken from https://data.pouet.net/")
	flag.BoolVar(&args.serve, "serve", false, "Start a server to serve REST API calls")
	flag.StringVar(&args.listen, "listen", "localhost:8000", "Address to listen to and to serve api calls from")
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
		create(db, args.pouet_prods, args.pouet_groups)
	}

	if args.serve {
		listen(db, args.listen)
	}
}
