package main

import (
	"flag"
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
