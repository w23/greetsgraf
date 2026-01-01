package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

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

func respondErrJson(w http.ResponseWriter, status int, err error) {
	response, jerr := json.Marshal(struct{ Error string }{Error: err.Error()})
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

func (c *Database) groupsFind(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	name := query.Get("name")
	if name == "" {
		respondJson(w, http.StatusOK, []int{})
		return
	}

	groups, err := c.FindGroups(name)
	if err != nil {
		// TODO proper error status
		respondJson(w, http.StatusInternalServerError, []int{})
		return
	}

	respondJson(w, http.StatusOK, &groups)
}

func (c *Database) findProd(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	name := query.Get("name")
	if name == "" {
		respondJson(w, http.StatusOK, []int{})
		return
	}

	prods, err := c.FindProds(name)
	if err != nil {
		// TODO proper error status
		respondJson(w, http.StatusInternalServerError, []int{})
		return
	}

	respondJson(w, http.StatusOK, &prods)
}

func (c *Database) prodGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pid := ctx.Value("prod_id")

	prod, err := c.GetProd(pid)
	if err != nil {
		// TODO proper error status
		respondJson(w, http.StatusInternalServerError, struct{}{})
		return
	}

	// TODO maybe it's better done through a custom marshaller ...
	type ResponseGroup struct {
		ID             uint
		Name           string
		Disambiguation string
	}

	type ResponseGreet struct {
		ID    uint
		Group ResponseGroup
		Note  string
	}

	response_prod := struct {
		ID         uint
		Name       string
		Year       int
		Month      int
		Day        int
		Video      string
		Rank       int
		VoteUp     int
		VotePig    int
		VoteDown   int
		Demozoo    int
		Screenshot string
		Groups     []ResponseGroup
		Greets     []ResponseGreet
	}{
		ID:         prod.ID,
		Name:       prod.Name,
		Year:       prod.Year,
		Month:      prod.Month,
		Day:        prod.Day,
		Video:      prod.Video,
		Rank:       prod.Rank,
		VoteUp:     prod.VoteUp,
		VotePig:    prod.VotePig,
		VoteDown:   prod.VoteDown,
		Demozoo:    prod.Demozoo,
		Screenshot: prod.Screenshot,
	}

	for i, _ := range prod.Groups {
		group := &prod.Groups[i]
		response_prod.Groups = append(response_prod.Groups, ResponseGroup{
			ID:             group.ID,
			Name:           group.Name,
			Disambiguation: group.Disambiguation,
		})
	}

	for i, _ := range prod.Greets {
		greet := &prod.Greets[i]
		group, err := c.GetGroup(greet.GreeteeID)
		if err != nil {
			log.Println(err)
			continue
		}
		response_prod.Greets = append(response_prod.Greets, ResponseGreet{
			ID:   greet.ID,
			Note: greet.Reference,
			Group: ResponseGroup{
				ID:             group.ID,
				Name:           group.Name,
				Disambiguation: group.Disambiguation,
			},
		})
	}

	respondJson(w, http.StatusOK, response_prod)
}

func (c *Database) prodGetGreets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	prod_id := ctx.Value("prod_id")

	var greets []struct {
		GreeteeID   uint
		GreeteeName string
		Reference   string
	}

	db := c.db.Table("greets").Select("greets.greetee_id as GreeteeID, groups.name as GreeteeName, greets.reference as Reference").Where("greets.prod_id = ?", prod_id).Joins("INNER JOIN groups ON groups.id = greets.greetee_id").Find(&greets)

	if db.Error != nil {
		respondErrJson(w, http.StatusInternalServerError, db.Error)
		return
	}

	respondJson(w, http.StatusOK, greets)
}

func (c *Database) groupGetGreeted(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	group_id := ctx.Value("group_id")

	var greets []Greet
	db := c.db.Find(&greets, "greetee_id = ?", group_id)

	if db.Error != nil {
		respondErrJson(w, http.StatusInternalServerError, db.Error)
		return
	}

	type ResponseItem struct {
		Prod      Prod
		Reference string
	}

	var response []ResponseItem

	for i := range greets {
		greet := &greets[i]
		var prod Prod
		c.db.Find(&prod, "id = ?", greet.ProdID).Association("Groups")
		c.db.Model(&prod).Association("Groups").Find(&prod.Groups)
		for j := range prod.Groups {
			prod.Groups[j].getCounts(c.db)
		}
		response = append(response, ResponseItem{
			Prod:      prod,
			Reference: greet.Reference,
		})
	}

	respondJson(w, http.StatusOK, &response)
}

func (c *Database) greetsCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProdId  uint
		GroupId uint
		Note    string
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
		respondJson(w, http.StatusOK, struct{ ID uint }{greet.ID})
	}
}

func (c *Database) greetsDelete(w http.ResponseWriter, r *http.Request) {
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
		respondJson(w, http.StatusOK, struct{ Rows int64 }{db.RowsAffected})
	}
}

func (c *Database) getStats(w http.ResponseWriter, r *http.Request) {
	var stats struct {
		TotalGreets     int64
		TotalProds      int64
		TotalGroups     int64
		ProdsWithGreets int64
		GreetedGroups   int64
	}

	c.db.Model(Greet{}).Count(&stats.TotalGreets)
	c.db.Model(Prod{}).Count(&stats.TotalProds)
	c.db.Model(Group{}).Count(&stats.TotalGroups)
	c.db.Model(Greet{}).Distinct("prod_id").Count(&stats.ProdsWithGreets)
	c.db.Model(Greet{}).Distinct("greetee_id").Count(&stats.GreetedGroups)

	respondJson(w, http.StatusOK, stats)
}

func (c *Database) groupsGreeted(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))

	var results []map[string]interface{}
	db := c.db.Model(Greet{}).Select("greets.greetee_id AS group_id, groups.name AS group_name, COUNT(DISTINCT greets.id) AS count").Joins("INNER JOIN groups ON groups.id = greets.greetee_id").Group("greets.greetee_id").Order("count DESC").Limit(limit).Find(&results)

	if db.Error != nil {
		respondErrJson(w, http.StatusInternalServerError, db.Error)
		return
	}

	respondJson(w, http.StatusOK, results)
}

func ProdContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		prod_id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			respondErrJson(w, http.StatusBadRequest, err)
			return
		}

		ctx := context.WithValue(r.Context(), "prod_id", prod_id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GroupContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		group_id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			respondErrJson(w, http.StatusBadRequest, err)
			return
		}

		ctx := context.WithValue(r.Context(), "group_id", group_id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func listen(db *gorm.DB, listen string, serve_static string) {
	ctx := Database{db}

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Timeout(10 * time.Second))

	r.Route("/v1", func(r chi.Router) {
		r.Get("/stats", ctx.getStats)

		r.Route("/groups", func(r chi.Router) {
			r.Get("/search", ctx.groupsFind)
			r.Get("/greeted", ctx.groupsGreeted)
			r.Route("/{id}", func(r chi.Router) {
				r.Use(GroupContext)
				//r.Get("/", ctx.groupGet)
				r.Get("/greets", ctx.groupGetGreeted)
			})
		})
		r.Route("/prods", func(r chi.Router) {
			r.Get("/search", ctx.findProd)
			r.Route("/{id}", func(r chi.Router) {
				r.Use(ProdContext)
				r.Get("/", ctx.prodGet)
				r.Get("/greets", ctx.prodGetGreets)
			})
		})

		r.Route("/greets", func(r chi.Router) {
			r.Post("/", ctx.greetsCreate)
			r.Route("/{id}", func(r chi.Router) {
				//r.Get("", ctx.greetsGet)
				//r.Patch("", ctx.greetsUpdate)
				r.Delete("/", ctx.greetsDelete)
			})
		})
	})

	if serve_static != "" {
		fs := http.FileServer(http.Dir(serve_static))
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			fs.ServeHTTP(w, r)
		})
	}

	log.Printf("Listening on %+v", listen)
	log.Fatal(http.ListenAndServe(listen, r))
}

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
		create(db, args.pouet_prods, args.pouet_groups)
		buildIndex(db)
	} else if args.index {
		buildIndex(db)
	}

	if args.serve {
		listen(db, args.listen, args.static)
	}
}
