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
	ProdsCount int64 `gorm:"-"`
	GreetsCount int64 `gorm:"-"`
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

func buildIndex(db *gorm.DB) {
	if err := db.Exec("CREATE VIRTUAL TABLE groups_fts USING fts5(name, id)").Error; err != nil {
		log.Fatalf("Failed to create FTS index for groups: %+v", err);
	}
	if err := db.Exec("INSERT INTO groups_fts (name, id) SELECT name, id FROM groups").Error; err != nil {
		log.Fatalf("Failed to populate FTS index for groups: %+v", err);
	}

	if err := db.Exec("CREATE VIRTUAL TABLE prods_fts USING fts5(name, id)").Error; err != nil {
		log.Fatalf("Failed to create FTS index for prods: %+v", err);
	}
	if err := db.Exec("INSERT INTO prods_fts (name, id) SELECT name, id FROM prods").Error; err != nil {
		log.Fatalf("Failed to populate FTS index for prods: %+v", err);
	}
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

func (c *Context) groupsFind(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	name := query.Get("name")
	if name == "" {
		respondJson(w, http.StatusOK, []int{})
		return
	}

	const limit = 10

	var groups []Group
	// FIXME FTS is very fragile. There are many inputs that will generate SQL errors. Let's just ignore any errors coming from it for now.
	/*db := */ c.db.Table("groups").Joins("INNER JOIN groups_fts ON groups_fts.id = groups.id").Where("groups_fts MATCH ?", name).Order("rank").Limit(limit).Find(&groups)
	// if db.Error == gorm.ErrRecordNotFound {
	// 	respondJson(w, http.StatusNotFound, struct{}{})
	// } else if db.Error != nil {
	// 	respondErrJson(w, http.StatusInternalServerError, db.Error)
	// } else
	{
		if len(groups) < limit {
			var like_groups []Group
			c.db.Limit(limit - len(groups)).Find(&like_groups, "name LIKE ?", "%" + name + "%")
			for i := range like_groups {
				gl := &like_groups[i]
				found := false
				for j := range groups {
					if groups[j].ID == gl.ID {
						found = true
						break
					}
				}
				if !found {
					groups = append(groups, *gl)
				}
			}
		}

		for i := range groups {
			g := &groups[i]
			g.ProdsCount = c.db.Model(g).Association("Prods").Count()
			c.db.Model(Greet{}).Where("greetee_id = ?", g.ID).Count(&g.GreetsCount)
		}

		respondJson(w, http.StatusOK, &groups)
	}
}

func (c *Context) findProd(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	name := query.Get("name")
	if name == "" {
		respondJson(w, http.StatusOK, []int{})
		return
	}

	db := c.db.Table("prods").Joins("INNER JOIN prods_fts ON prods_fts.id = prods.id").Where("prods_fts MATCH ?", name).Order("prods_fts.rank")

	const limit = 10

	var prods []Prod
	db = db.Preload("Groups").Limit(limit).Find(&prods)
	// FIXME FTS is very fragile. There are many inputs that will generate SQL errors. Let's just ignore any errors coming from it for now.
	//if db.Error == gorm.ErrRecordNotFound {
	// respondJson(w, http.StatusNotFound, struct{}{})
	// } else if db.Error != nil {
	// 	respondErrJson(w, http.StatusInternalServerError, db.Error)
	//} else
	{
		if len(prods) < limit {
			var like_prods []Prod
			c.db.Preload("Groups").Limit(limit - len(prods)).Find(&like_prods, "name LIKE ?", "%" + name + "%")
			for i := range like_prods {
				gl := &like_prods[i]
				found := false
				for j := range prods {
					if prods[j].ID == gl.ID {
						found = true
						break
					}
				}
				if !found {
					prods = append(prods, *gl)
				}
			}
		}
		respondJson(w, http.StatusOK, &prods)
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

		// TODO maybe it's better done through a custom marshaller ...
		type ResponseGroup struct {
			ID uint
			Name string
			Disambiguation string
		}

		type ResponseGreet struct {
			ID uint
			Group ResponseGroup
			Note string
		}

		response_prod := struct {
			ID uint
			Name string
			Year int
			Month int
			Day int
			Video string
			Rank int
			VoteUp int
			VotePig int
			VoteDown int
			Demozoo int
			Screenshot string
			Groups []ResponseGroup
			Greets []ResponseGreet
		} {
			ID: prod.ID,
			Name: prod.Name,
			Year: prod.Year,
			Month: prod.Month,
			Day: prod.Day,
			Video: prod.Video,
			Rank: prod.Rank,
			VoteUp: prod.VoteUp,
			VotePig: prod.VotePig,
			VoteDown: prod.VoteDown,
			Demozoo: prod.Demozoo,
			Screenshot: prod.Screenshot,
		}

		for i, _ := range prod.Groups {
			group := &prod.Groups[i]
			response_prod.Groups = append(response_prod.Groups, ResponseGroup{
				ID: group.ID,
				Name: group.Name,
				Disambiguation: group.Disambiguation,
			})
		}

		for i, _ := range prod.Greets {
			greet := &prod.Greets[i]
			var group Group
			c.db.Find(&group, "ID = ?", greet.GreeteeID)
			response_prod.Greets = append(response_prod.Greets, ResponseGreet{
				ID: greet.ID,
				Note: greet.Reference,
				Group: ResponseGroup{
					ID: group.ID,
					Name: group.Name,
					Disambiguation: group.Disambiguation,
				},
			})
		}

		respondJson(w, http.StatusOK, response_prod)
	}
}

func (c *Context) greetsCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProdId uint
		GroupId uint
		Note string
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
			Reference: body.Note,
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

func (c *Context) getStats(w http.ResponseWriter, r *http.Request) {
	var stats struct{
		TotalGreets int64
		TotalProds int64
		TotalGroups int64
		ProdsWithGreets int64
		GreetedGroups int64
	}

	c.db.Model(Greet{}).Count(&stats.TotalGreets)
	c.db.Model(Prod{}).Count(&stats.TotalProds)
	c.db.Model(Group{}).Count(&stats.TotalGroups)
	c.db.Model(Greet{}).Distinct("prod_id").Count(&stats.ProdsWithGreets)
	c.db.Model(Greet{}).Distinct("greetee_id").Count(&stats.GreetedGroups)

	respondJson(w, http.StatusOK, stats)
}

func (c *Context) groupsGreeted(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))

	var results []map[string]interface{}
	db := c.db.Model(Greet{}).Select("greets.greetee_id AS group_id, groups.name AS group_name, COUNT(DISTINCT greets.id) AS count").Joins("INNER JOIN groups ON groups.id = greets.greetee_id").Group("greets.greetee_id").Order("count DESC").Limit(limit).Find(&results)

	if db.Error != nil {
		respondErrJson(w, http.StatusInternalServerError, db.Error)
		return;
	}

	respondJson(w, http.StatusOK, results)
}

func listen(db *gorm.DB, listen string, serve_static string) {
	ctx := Context{db}

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Timeout(10 * time.Second))

	r.Route("/v1", func (r chi.Router) {
		r.Get("/stats", ctx.getStats)

		r.Route("/groups", func (r chi.Router) {
			r.Get("/search", ctx.groupsFind)
			r.Get("/greeted", ctx.groupsGreeted)
			//r.Get("/{id}", ctx.getGroup)
		})
		r.Route("/prods", func (r chi.Router) {
			r.Get("/search", ctx.findProd)
			r.Route("/{id}", func (r chi.Router) {
				r.Get("/", ctx.prodGet)
				//r.Get("/greets", ctx.prodGetGreets)
			})
		})

		r.Route("/greets", func (r chi.Router) {
			r.Post("/", ctx.greetsCreate)
			r.Route("/{id}", func (r chi.Router) {
				//r.Get("", ctx.greetsGet)
				//r.Patch("", ctx.greetsUpdate)
				r.Delete("/", ctx.greetsDelete)
			})
		})
	})

	if serve_static != "" {
		fs := http.FileServer(http.Dir(serve_static))
		r.Get("/*", func (w http.ResponseWriter, r *http.Request) {
			fs.ServeHTTP(w, r)
		})
	}

	log.Printf("Listening on %+v", listen)
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
	static string
	index bool
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
		create(db, args.pouet_prods, args.pouet_groups)
		buildIndex(db)
	} else if args.index {
		buildIndex(db)
	}

	if args.serve {
		listen(db, args.listen, args.static)
	}
}
