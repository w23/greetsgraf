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
	"net/http"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"strings"
)

type Group struct {
	ID uint `gorm:"primaryKey"`
	Name string `gorm:"index"`
	Disambiguation string `gorm:"index"`
	Prods []Prod `gorm:"many2many:group_prods;"`
	//Greeted []Greet `gorm:"many2many:group_greeted;"`
	//Greets []Greet `gorm:"many2many:group_greets;"`
}

type Prod struct {
	ID uint `gorm:"primaryKey"`
	Name string `gorm:"index"`
	Year int `gorm:"index"`
	Month int `gorm:"index"`
	Day int `gorm:"index"`
	Video string
	Rank int
	VoteUp int
	VotePig int
	VoteDown int
	Demozoo int
	Screenshot string
	// TODO: credits
	Groups []Group `gorm:"many2many:group_prods;"`
	Greets []Greet
}

type Greet struct {
	gorm.Model
	UserID uint `gorm:"index"`
	Reference string
	// ??? GroupName string
	ProdID uint `gorm:"uniqueIndex:greets_prod_group"`
	GreeteeID uint `gorm:"uniqueIndex:greets_prod_group"` //;many2many:group_greeted;"`
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

func ContainsInsensitive(a, b string) bool {
	return strings.Contains(strings.ToLower(a), strings.ToLower(b))
}

func create(db *gorm.DB, prodsfile string, groupsfile string) {
	if prodsfile == "" || groupsfile == "" {
		flag.Usage()
		log.Fatal("When creating a new db, pouet data dumps are needed\n")
	}

	db.AutoMigrate(&Group{})
	db.AutoMigrate(&Prod{})
	db.AutoMigrate(&Greet{})

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
				ID: uint(pouet_id),
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

		prods_array := (prods["prods"]).([]interface{})
		num_prods := len(prods_array)

		tx := db.Begin()
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

			rank, _ := strconv.Atoi(prod["rank"].(string))
			voteup, _ := strconv.Atoi(prod["voteup"].(string))
			votepig, _ := strconv.Atoi(prod["votepig"].(string))
			votedown, _ := strconv.Atoi(prod["votedown"].(string))

			var demozoo int
			if json_demozoo, have_demozoo := prod["demozoo"]; have_demozoo && json_demozoo != nil {
				demozoo, _ = strconv.Atoi(json_demozoo.(string))
			}

			var video string
			if dlinks, have := prod["downloadLinks"]; have {
				array := dlinks.([]interface{})
				for _, jlink := range array {
					link := jlink.(map[string]interface{})
					ltype := strings.ToLower(link["type"].(string))
					if strings.Contains(ltype, "youtube") {
						video = link["link"].(string)
						break
					}
					if strings.Contains(ltype, "vimeo") {
						video = link["link"].(string)
						break
					}
				}
			}

			var screenshot string
			if shot, found := prod["screenshot"]; found && shot != nil {
				screenshot = shot.(string)
			}

			dbprod := Prod{
				ID: uint(pid),
				Name: name,
				Year: date.Year(),
				Month: int(date.Month()),
				Day: date.Day(),
				Rank: rank,
				VoteUp: voteup,
				VoteDown: votedown,
				VotePig: votepig,
				Demozoo: demozoo,
				Video: video,
				Screenshot: screenshot,
			}

			// Associate with groups
			jgroups := prod["groups"].([]interface{})
			for _, jgroup := range jgroups {
				group := jgroup.(map[string]interface{})
				gid, err := strconv.Atoi(group["id"].(string))
				if err != nil {
					log.Printf("Cannot parse '%+v' as id: %+v", group["id"], err)
					continue
				}

				dbprod.Groups = append(dbprod.Groups, Group{ID: uint(gid)})
			}

			tx.Create(&dbprod)

			if (i + 1) % 1000 == 0 {
				log.Printf("Processed %d / %d", i + 1, num_prods)
			}
		}

		tx.Commit()
	}

	log.Printf("Import done.")
}

func respondErrJson(w http.ResponseWriter, status int, err error) {
	response, jerr := json.Marshal(struct{Error string}{Error: err.Error()})
	if jerr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(response))
}

func respondJson(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		respondErrJson(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(response))
}

type Context struct {
	db *gorm.DB
}

func (c *Context) findGroup(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	name := query.Get("name")
	if name == "" {
		respondJson(w, http.StatusOK, []int{})
		return
	}

	var groups []Group
	c.db.Where("name LIKE ?", "%"+name+"%").Limit(10).Find(&groups)
	respondJson(w, http.StatusOK, &groups)
}

func (c *Context) findProd(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	name := query.Get("name")
	if name == "" {
		respondJson(w, http.StatusOK, []int{})
		return
	}

	db := c.db.Where("name LIKE ?", "%"+name+"%")

	var prods []Prod
	db = db.Preload("Groups").Limit(10).Find(&prods)
	if db.Error == gorm.ErrRecordNotFound {
		respondJson(w, http.StatusNotFound, struct{}{})
	} else if db.Error != nil {
		respondErrJson(w, http.StatusInternalServerError, db.Error)
	} else {
		respondJson(w, http.StatusOK, prods)
	}
}

func (c *Context) prodGet(w http.ResponseWriter, r *http.Request) {
	pid, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondErrJson(w, http.StatusBadRequest, err)
		return
	}

	var prod Prod
	db := c.db.Find(&prod, "id = ?", pid)
	if db.Error == gorm.ErrRecordNotFound {
		respondJson(w, http.StatusNotFound, struct{}{})
	} else if db.Error != nil {
		respondErrJson(w, http.StatusInternalServerError, db.Error)
	} else {
		c.db.Model(&prod).Association("Groups").Find(&prod.Groups)
		c.db.Model(&prod).Association("Greets").Find(&prod.Greets)
		respondJson(w, http.StatusOK, prod)
	}
}

// func (c *Context) greetsSearch(w http.ResponseWriter, r *http.Request) {
// 	query := r.URL.Query()
//
// 	db := c.db
// 	prod_id, err := strconv.Atoi(query.Get("prod"))
// 	if err == nil {
// 		db = db.Where("prod_id = ?", prod_id)
// 	}
//
// 	greetee_id, err := strconv.Atoi(query.Get("greetee"))
// 	if err == nil {
// 		db = db.Where("greetee_id = ?", greetee_id)
// 	}
//
// 	// group_id, err := strconv.Atoi(query.Get("greeter"))
// 	// if err == nil {
// 	// 	db = db.Where("greetee_id = ?", group_id)
// 	// }
//
// 	var greets []Greet
// 	db = db.Find(&greets)
// 	log.Printf("%+v", greets)
// 	err = db.Error
// 	if err != nil {
// 		log.Printf("Error: %+v", err)
// 		respondErrJson(w, http.StatusInternalServerError, err)
// 		return
// 	}
// 	respondJson(w, http.StatusOK, greets)
// }

func (c *Context) greetsCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProdId uint
		GroupId uint
		Reference string
	}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		respondErrJson(w, http.StatusBadRequest, err)
		return
	}

	{
		tx := c.db.Begin()
		defer tx.Rollback()

		var prod Prod
		if err := tx.Find(&prod, "id = ?", body.ProdId).Error; err != nil {
			// TODO status not found if errrecordnotfound
			respondErrJson(w, http.StatusBadRequest, err)
			return
		}

		greet := Greet{
			Reference: body.Reference,
			GreeteeID: body.GroupId,
		}

		if err := tx.Model(&prod).Association("Greets").Append(&greet); err != nil {
			// TODO what errors might be here?
			respondErrJson(w, http.StatusBadRequest, err)
			return
		}

		if err := tx.Commit().Error; err != nil {
			// TODO what errors might be here?
			respondErrJson(w, http.StatusInternalServerError, err)
			return
		}
		respondJson(w, http.StatusOK, struct{ID uint}{greet.ID})
	}
}

func (c *Context) greetsDelete(w http.ResponseWriter, r *http.Request) {
	greet_id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondErrJson(w, http.StatusBadRequest, err)
		return
	}

	db := c.db.Unscoped().Delete(&Greet{}, "id = ?", greet_id)
	if db.Error == gorm.ErrRecordNotFound {
		respondJson(w, http.StatusNotFound, struct{}{})
	} else if db.Error != nil {
		respondErrJson(w, http.StatusInternalServerError, db.Error)
	} else {
		respondJson(w, http.StatusOK, struct{Rows int64}{db.RowsAffected})
	}
}

func listen(db *gorm.DB, listen string) {
	ctx := Context{db}

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/v1/groups/search", ctx.findGroup)
	r.Route("/v1/prods", func (r chi.Router) {
		r.Get("/search", ctx.findProd)
		r.Route("/{id}", func (r chi.Router) {
			r.Get("/", ctx.prodGet)
			//r.Get("/greets", ctx.prodGetGreets)
		})
	})

	r.Route("/v1/greets", func (r chi.Router) {

	// TODO Need to cover:
	// - all greets from group id
	//	- constraints:
	//		- time
	//		-	to group(s)
	//	- sort by:
	//		- time
	//    - count ?
	// - all greets from prod types

		//r.Get("/search", ctx.greetsSearch)

		r.Post("/", ctx.greetsCreate)

		r.Route("/{id}", func (r chi.Router) {
			//r.Get("", ctx.greetsGet)
			//r.Patch("", ctx.greetsUpdate)
			r.Delete("/", ctx.greetsDelete)
		})
	})

	r.Get("/", func (w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "index.html") })

	log.Fatal(http.ListenAndServe(listen, r))
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
