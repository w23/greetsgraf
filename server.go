package main

import (
	"context"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

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

	greets, err := c.GetProdGreets(prod_id)
	if err != nil {
		respondErrJson(w, http.StatusInternalServerError, err)
		return
	}

	respondJson(w, http.StatusOK, greets)
}

func (c *Database) groupGetGreeted(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	group_id := ctx.Value("group_id")

	greets, err := c.GetGroupGreets(group_id)
	if err != nil {
		respondErrJson(w, http.StatusInternalServerError, err)
		return
	}

	respondJson(w, http.StatusOK, &greets)
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

	id, err := c.Greet(body.ProdId, body.GroupId, body.Note)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate greet") {
			respondErrJson(w, http.StatusBadRequest, err)
			return
		}
		respondErrJson(w, http.StatusInternalServerError, err)
		return
	}

	respondJson(w, http.StatusOK, struct{ ID uint }{id})
}

func (c *Database) greetsDelete(w http.ResponseWriter, r *http.Request) {
	greet_id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondErrJson(w, http.StatusBadRequest, err)
		return
	}

	removed, err := c.DeleteGreet(uint(greet_id))
	if err != nil {
		respondErrJson(w, http.StatusInternalServerError, err)
		return
	}

	if !removed {
		respondJson(w, http.StatusNotFound, struct{ Rows int64 }{Rows: 0})
		return
	}

	respondJson(w, http.StatusOK, struct{ Rows int64 }{Rows: 1})
}

func (c *Database) getStats(w http.ResponseWriter, r *http.Request) {
	stats := c.GetStats()
	respondJson(w, http.StatusOK, stats)
}

func (c *Database) groupsGreeted(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limitStr := query.Get("limit")
	limit := 20 // default
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil {
			limit = parsed
		}
	}

	results, err := c.GetMostGreetedGroups(limit)

	if err != nil {
		respondErrJson(w, http.StatusInternalServerError, err)
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

func Server(db Database, serve_static string) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Timeout(10 * time.Second))

	r.Route("/v1", func(r chi.Router) {
		r.Get("/stats", db.getStats)

		r.Route("/groups", func(r chi.Router) {
			r.Get("/search", db.groupsFind)
			r.Get("/greeted", db.groupsGreeted)
			r.Route("/{id}", func(r chi.Router) {
				r.Use(GroupContext)
				//r.Get("/", db.groupGet)
				r.Get("/greets", db.groupGetGreeted)
			})
		})
		r.Route("/prods", func(r chi.Router) {
			r.Get("/search", db.findProd)
			r.Route("/{id}", func(r chi.Router) {
				r.Use(ProdContext)
				r.Get("/", db.prodGet)
				r.Get("/greets", db.prodGetGreets)
			})
		})

		r.Route("/greets", func(r chi.Router) {
			r.Post("/", db.greetsCreate)
			r.Route("/{id}", func(r chi.Router) {
				//r.Get("", db.greetsGet)
				//r.Patch("", db.greetsUpdate)
				r.Delete("/", db.greetsDelete)
			})
		})
	})

	if serve_static != "" {
		fs := http.FileServer(http.Dir(serve_static))
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			fs.ServeHTTP(w, r)
		})
	}

	return r
}

func listen(db Database, listen_addr string, serve_static string) {
	r := Server(db, serve_static)

	log.Printf("Listening on %+v", listen_addr)
	log.Fatal(http.ListenAndServe(listen_addr, r))
}
