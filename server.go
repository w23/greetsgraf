package main

import (
	"context"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log"
	"net/http"
	"strconv"
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

func (p *Pouet) groupsFind(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	name := query.Get("name")
	if name == "" {
		respondJson(w, http.StatusOK, []int{})
		return
	}

	groups, err := p.FindGroups(name)
	if err != nil {
		// TODO proper error status
		respondJson(w, http.StatusInternalServerError, []int{})
		return
	}

	respondJson(w, http.StatusOK, &groups)
}

func (p *Pouet) findProd(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	name := query.Get("name")
	if name == "" {
		respondJson(w, http.StatusOK, []int{})
		return
	}

	prods, err := p.FindProds(name)
	if err != nil {
		// TODO proper error status
		respondJson(w, http.StatusInternalServerError, []int{})
		return
	}

	respondJson(w, http.StatusOK, &prods)
}

func (p *Pouet) prodGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pid := ctx.Value("prod_id")

	prod, err := p.GetProd(pid)
	if err != nil {
		respondJson(w, http.StatusInternalServerError, struct{}{})
		return
	}

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

	type ResponseGroupWithID struct {
		ID             uint
		Name           string
		Disambiguation string
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

	for i := range prod.Groups {
		group := &prod.Groups[i]
		response_prod.Groups = append(response_prod.Groups, ResponseGroup{
			ID:             group.ID,
			Name:           group.Name,
			Disambiguation: group.Disambiguation,
		})
	}

	greets, err := p.GetProdGreets(r, pid)
	if err != nil {
		log.Println(err)
	} else {
		for _, greet := range greets {
			response_prod.Greets = append(response_prod.Greets, ResponseGreet{
				Note: greet.Reference,
				Group: ResponseGroup{
					ID:             greet.GreeteeID,
					Name:           greet.GreeteeName,
					Disambiguation: "",
				},
			})
		}
	}

	respondJson(w, http.StatusOK, response_prod)
}

func (p *Pouet) prodGetGreets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	prod_id := ctx.Value("prod_id")

	greets, err := p.GetProdGreets(r, prod_id)
	if err != nil {
		respondErrJson(w, http.StatusInternalServerError, err)
		return
	}

	respondJson(w, http.StatusOK, greets)
}

func (p *Pouet) groupGetGreeted(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	group_id := ctx.Value("group_id")

	greets, err := p.GetGroupGreets(r, group_id)
	if err != nil {
		respondErrJson(w, http.StatusInternalServerError, err)
		return
	}

	respondJson(w, http.StatusOK, &greets)
}

func (g *Greets) greetsCreate(w http.ResponseWriter, r *http.Request) {
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

	id, err := g.Greet(body.ProdId, body.GroupId, body.Note)
	if err != nil {
		respondErrJson(w, http.StatusInternalServerError, err)
		return
	}

	respondJson(w, http.StatusOK, struct{ ID uint }{id})
}

func (g *Greets) greetsDelete(w http.ResponseWriter, r *http.Request) {
	greet_id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondErrJson(w, http.StatusBadRequest, err)
		return
	}

	removed, err := g.DeleteGreet(uint(greet_id))
	if err != nil {
		respondErrJson(w, http.StatusInternalServerError, err)
		return
	}

	if !removed {
		respondJson(w, http.StatusNotFound, struct{}{})
	}

	respondJson(w, http.StatusOK, struct{}{})
}

func (p *Pouet) getStats(w http.ResponseWriter, r *http.Request) {
	stats := p.GetStats(r)
	respondJson(w, http.StatusOK, stats)
}

func (p *Pouet) groupsGreeted(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))

	results, err := p.GetMostGreetedGroups(r, limit)

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

func listen(pouetDB *Pouet, greetsDB *Greets, listen string, serve_static string) {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Timeout(10 * time.Second))

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), "writableDB", greetsDB)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	r.Route("/v1", func(r chi.Router) {
		r.Get("/stats", pouetDB.getStats)

		r.Route("/groups", func(r chi.Router) {
			r.Get("/search", pouetDB.groupsFind)
			r.Get("/greeted", pouetDB.groupsGreeted)
			r.Route("/{id}", func(r chi.Router) {
				r.Use(GroupContext)
				//r.Get("/", pouetDB.groupGet)
				r.Get("/greets", pouetDB.groupGetGreeted)
			})
		})
		r.Route("/prods", func(r chi.Router) {
			r.Get("/search", pouetDB.findProd)
			r.Route("/{id}", func(r chi.Router) {
				r.Use(ProdContext)
				r.Get("/", pouetDB.prodGet)
				r.Get("/greets", pouetDB.prodGetGreets)
			})
		})

		r.Route("/greets", func(r chi.Router) {
			r.Post("/", greetsDB.greetsCreate)
			r.Route("/{id}", func(r chi.Router) {
				//r.Get("", greetsDB.greetsGet)
				//r.Patch("", greetsDB.greetsUpdate)
				r.Delete("/", greetsDB.greetsDelete)
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
