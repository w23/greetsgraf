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
	"time"
)

type Group struct {
	ID int `gorm:"primaryKey"`
	Name string
	Disambiguation string
	Prods []*Prod `gorm:"many2many:group_prods;"`
}

type Prod struct {
	ID int `gorm:"primaryKey"`
	Name string
	Year int
	Month int
	Day int
	Groups []*Group `gorm:"many2many:group_prods;"`
}

// type Greet struct {
// 	gorm.Model
// 	GreeterID *Group
// 	GreeteeID *Group
// 	ProdID *Prod
// }

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
	db.AutoMigrate(&Prod{})

	log.Printf("Importing groups...")

	{
		groups, err := readJsonGz(groupsfile)
		if err != nil {
			log.Fatalf("Unable to read groups from file %s: %v", groupsfile, err)
		}

		tx := db.Begin()

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
				ID: int(pouet_id),
				Name: name,
				Disambiguation: disambiguation,
			}

			tx.Create(&dbgroup)
		}

		tx.Commit()
	}

	log.Printf("Importing prods...")

	{
		prods, err := readJsonGz(prodsfile)
		if err != nil {
			log.Fatalf("Unable to read prods from file %s: %v", groupsfile, err)
		}

		log.Printf("Loaded prods json into memory...")

		tx := db.Begin()

		prods_array := (prods["prods"]).([]interface{})
		num_prods := len(prods_array)
		for i, iprod := range prods_array {
			prod := iprod.(map[string]interface{})
			pid, err := strconv.Atoi(prod["id"].(string))
			if err != nil {
				log.Printf("wtf id %s", prod["id"])
				continue
			}

			name := prod["name"].(string)
			jdate, found := prod["releaseDate"]

			var date time.Time

			if found && jdate != nil {
				date_string := jdate.(string)
				date, err = time.Parse("2006-01-02", date_string)
				// TODO: for missing/invalid dates try to parse manually, or refer to party_year
				if err != nil {
					log.Printf("Prod %d:%s: cannot parse '%+v' as date: %+v", pid, prod["name"], date_string, err)
					continue
				}
			} else {
				log.Printf("Prod %v:%v has no date", prod["id"], name)
			}

			dbprod := Prod{
				ID: pid,
				Name: name,
				Year: date.Year(),
				Month: int(date.Month()),
				Day: date.Day(),
			}

			tx.Create(&dbprod)

			// Associate with groups
			jgroups := prod["groups"].([]interface{})
			var groups []Group
			for _, jgroup := range jgroups {
				group := jgroup.(map[string]interface{})
				gid, err := strconv.Atoi(group["id"].(string))
				if err != nil {
					log.Printf("Cannot parse '%+v' as id: %+v", group["id"], err)
					continue
				}

				groups = append(groups, Group{ID: gid})
			}

			if len(groups) > 0 {
				tx.Model(&dbprod).Association("Groups").Append(groups)
			}

			if (i + 1) % 1000 == 0 {
				log.Printf("Processed %d / %d", i + 1, num_prods)
			}
		}

		tx.Commit()
	}

	log.Printf("Import done.")
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
