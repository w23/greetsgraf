package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Group struct {
	ID             uint   `gorm:"primaryKey"`
	Name           string `gorm:"index"`
	Disambiguation string `gorm:"index"`
	Prods          []Prod `gorm:"many2many:group_prods;"`
	//Greeted []Greet `gorm:"many2many:group_greeted;"`
	//Greets []Greet `gorm:"many2many:group_greets;"`
	ProdsCount  int64 `gorm:"-"`
	GreetsCount int64 `gorm:"-"`
}

type Prod struct {
	ID         uint   `gorm:"primaryKey"`
	Name       string `gorm:"index"`
	Year       int    `gorm:"index"`
	Month      int    `gorm:"index"`
	Day        int    `gorm:"index"`
	Video      string
	Rank       int
	VoteUp     int
	VotePig    int
	VoteDown   int
	Demozoo    int
	Screenshot string
	// TODO: credits
	Groups []Group `gorm:"many2many:group_prods;"`
	Greets []Greet
}

type Greet struct {
	gorm.Model
	UserID    uint `gorm:"index"`
	Reference string
	// ??? GroupName string
	ProdID    uint `gorm:"uniqueIndex:greets_prod_group"`
	GreeteeID uint `gorm:"uniqueIndex:greets_prod_group"` //;many2many:group_greeted;"`
}

type ProdGreet struct {
	GreeteeID   uint
	GreeteeName string
	Reference   string
}

type GroupGreet struct {
	Prod      Prod
	Reference string
}

type Database struct {
	db *gorm.DB
}

func DatabaseOpen(datafile string) (db *gorm.DB, err error) {
	db, err = gorm.Open(sqlite.Open(datafile), &gorm.Config{})
	return
}

func buildIndex(db *gorm.DB) {
	if err := db.Exec("CREATE VIRTUAL TABLE groups_fts USING fts5(name, id)").Error; err != nil {
		log.Fatalf("Failed to create FTS index for groups: %+v", err)
	}
	if err := db.Exec("INSERT INTO groups_fts (name, id) SELECT name, id FROM groups").Error; err != nil {
		log.Fatalf("Failed to populate FTS index for groups: %+v", err)
	}

	if err := db.Exec("CREATE VIRTUAL TABLE prods_fts USING fts5(name, id)").Error; err != nil {
		log.Fatalf("Failed to create FTS index for prods: %+v", err)
	}
	if err := db.Exec("INSERT INTO prods_fts (name, id) SELECT name, id FROM prods").Error; err != nil {
		log.Fatalf("Failed to populate FTS index for prods: %+v", err)
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
				ID:             uint(pouet_id),
				Name:           name,
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
				ID:         uint(pid),
				Name:       name,
				Year:       date.Year(),
				Month:      int(date.Month()),
				Day:        date.Day(),
				Rank:       rank,
				VoteUp:     voteup,
				VoteDown:   votedown,
				VotePig:    votepig,
				Demozoo:    demozoo,
				Video:      video,
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

			if (i+1)%1000 == 0 {
				log.Printf("Processed %d / %d", i+1, num_prods)
			}
		}

		tx.Commit()
	}

	log.Printf("Import done.")
}

func (db *Database) FindGroups(name string) ([]Group, error) {
	const limit = 10

	var groups []Group
	// FIXME FTS is very fragile. There are many inputs that will generate SQL errors. Let's just ignore any errors coming from it for now.
	/*db := */
	db.db.Table("groups").Joins("INNER JOIN groups_fts ON groups_fts.id = groups.id").Where("groups_fts MATCH ?", name).Order("rank").Limit(limit).Find(&groups)
	// if db.Error == gorm.ErrRecordNotFound {
	// 	respondJson(w, http.StatusNotFound, struct{}{})
	// } else if db.Error != nil {
	// 	respondErrJson(w, http.StatusInternalServerError, db.Error)
	// } else
	if len(groups) < limit {
		var like_groups []Group
		db.db.Limit(limit-len(groups)).Find(&like_groups, "name LIKE ?", "%"+name+"%")
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
		groups[i].getCounts(db.db)
	}

	return groups, nil
}

func (db *Database) FindProds(name string) ([]Prod, error) {
	query := db.db.Table("prods").Joins("INNER JOIN prods_fts ON prods_fts.id = prods.id").Where("prods_fts MATCH ?", name).Order("prods_fts.rank")

	const limit = 10

	var prods []Prod
	query = query.Preload("Groups").Limit(limit).Find(&prods)
	// FIXME FTS is very fragile. There are many inputs that will generate SQL errors. Let's just ignore any errors coming from it for now.
	//if db.Error == gorm.ErrRecordNotFound {
	// respondJson(w, http.StatusNotFound, struct{}{})
	// } else if db.Error != nil {
	// 	respondErrJson(w, http.StatusInternalServerError, db.Error)
	//} else
	if len(prods) < limit {
		var like_prods []Prod
		db.db.Preload("Groups").Limit(limit-len(prods)).Find(&like_prods, "name LIKE ?", "%"+name+"%")
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

	return prods, nil
}

func (g *Group) getCounts(db *gorm.DB) {
	g.ProdsCount = db.Model(g).Association("Prods").Count()
	db.Model(Greet{}).Where("greetee_id = ?", g.ID).Count(&g.GreetsCount)
}

// TODO proper type for pid
func (db *Database) GetProd(pid any) (Prod, error) {
	var prod Prod
	query := db.db.Find(&prod, "id = ?", pid)
	if query.Error == gorm.ErrRecordNotFound {
		return prod, fmt.Errorf("not found")
	} else if query.Error != nil {
		return prod, fmt.Errorf("unknown db error: %w", query.Error)
	}

	db.db.Model(&prod).Association("Groups").Find(&prod.Groups)
	db.db.Model(&prod).Association("Greets").Find(&prod.Greets)

	return prod, nil
}

// TODO proper type for groupID
func (db *Database) GetGroup(groupID any) (Group, error) {
	var group Group
	query := db.db.Find(&group, "ID = ?", groupID)

	if query.Error != nil {
		return group, fmt.Errorf("find group id=%u: %w", groupID, query.Error)
	}

	return group, nil
}

func (db *Database) GetProdGreets(prodID any) ([]ProdGreet, error) {
	var greets []ProdGreet
	query := db.db.Table("greets").Select("greets.greetee_id as GreeteeID, groups.name as GreeteeName, greets.reference as Reference").Where("greets.prod_id = ?", prodID).Joins("INNER JOIN groups ON groups.id = greets.greetee_id").Find(&greets)

	if query.Error != nil {
		return []ProdGreet{}, fmt.Errorf("get greets for prod=%v: %w", prodID, query.Error)
	}

	return greets, nil
}

func (db *Database) GetGroupGreets(groupID any) ([]GroupGreet, error) {
	var raw_greets []Greet
	query := db.db.Find(&raw_greets, "greetee_id = ?", groupID)

	if query.Error != nil {
		return []GroupGreet{}, fmt.Errorf("get greets for greetee_id=%v: %w", groupID, query.Error)
	}

	var greets []GroupGreet

	for i := range raw_greets {
		raw_greet := &raw_greets[i]
		var prod Prod
		db.db.Find(&prod, "id = ?", raw_greet.ProdID).Association("Groups")
		db.db.Model(&prod).Association("Groups").Find(&prod.Groups)
		for j := range prod.Groups {
			prod.Groups[j].getCounts(db.db)
		}
		greets = append(greets, GroupGreet{
			Prod:      prod,
			Reference: raw_greet.Reference,
		})
	}

	return greets, nil
}

type DatabaseStats struct {
	TotalGreets     int64
	TotalProds      int64
	TotalGroups     int64
	ProdsWithGreets int64
	GreetedGroups   int64
}

func (db *Database) GetStats() DatabaseStats {
	var stats DatabaseStats
	db.db.Model(Greet{}).Count(&stats.TotalGreets)
	db.db.Model(Prod{}).Count(&stats.TotalProds)
	db.db.Model(Group{}).Count(&stats.TotalGroups)
	db.db.Model(Greet{}).Distinct("prod_id").Count(&stats.ProdsWithGreets)
	db.db.Model(Greet{}).Distinct("greetee_id").Count(&stats.GreetedGroups)
	return stats
}

func (db *Database) Greet(prodID uint, groupID uint, note string) (uint, error) {
	tx := db.db.Begin()
	defer tx.Rollback()

	var prod Prod
	if err := tx.Find(&prod, "id = ?", prodID).Error; err != nil {
		// TODO status not found if errrecordnotfound
		return 0, fmt.Errorf("find prod id=%v: %w", prodID, err)
	}

	greet := Greet{
		Reference: note,
		GreeteeID: groupID,
	}

	if err := tx.Model(&prod).Association("Greets").Append(&greet); err != nil {
		// TODO what errors might be here?
		return 0, fmt.Errorf("associate greets: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		// TODO what errors might be here?
		return 0, fmt.Errorf("tx commit: %w", err)
	}

	return greet.ID, nil
}

func (db *Database) DeleteGreet(greetID uint) (bool, error) {
	query := db.db.Unscoped().Delete(&Greet{}, "id = ?", greetID)
	if query.Error == gorm.ErrRecordNotFound {
		return false, nil
	} else if query.Error != nil {
		return false, fmt.Errorf("delete greet=%u: %w", greetID, query.Error)
	}

	if query.RowsAffected == 0 {
		return false, nil
	}

	return true, nil
}

func (db *Database) GetMostGreetedGroups(limit int) ([]map[string]any, error) {
	var results []map[string]any
	query := db.db.Model(Greet{}).Select("greets.greetee_id AS group_id, groups.name AS group_name, COUNT(DISTINCT greets.id) AS count").Joins("INNER JOIN groups ON groups.id = greets.greetee_id").Group("greets.greetee_id").Order("count DESC").Limit(limit).Find(&results)

	if query.Error != nil {
		return []map[string]any{}, fmt.Errorf("get most %d greeted groups: %w", limit, query.Error)
	}

	return results, nil
}
