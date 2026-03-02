package main

import (
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Group struct {
	ID             uint
	Name           string
	Disambiguation string
	ProdsCount     int64
	GreetsCount    int64
}

type Prod struct {
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
	Groups     []Group
}

type PouetDatabase struct {
	db *sql.DB
}

func PouetDatabaseOpen(filename string) (PouetDatabase, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return PouetDatabase{}, fmt.Errorf("open pouet database file %s: %w", filename, err)
	}

	return PouetDatabase{db: db}, nil
}

func (db *PouetDatabase) createPouetTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS groups (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		disambiguation TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_groups_name ON groups(name);
	CREATE INDEX IF NOT EXISTS idx_groups_disambiguation ON groups(disambiguation);

	CREATE TABLE IF NOT EXISTS prods (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		year INTEGER,
		month INTEGER,
		video TEXT,
		rank INTEGER,
		voteup INTEGER,
		votepig INTEGER,
		votedown INTEGER,
		demozoo INTEGER,
		screenshot TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_prods_name ON prods(name);
	CREATE INDEX IF NOT EXISTS idx_prods_year ON prods(year);
	CREATE INDEX IF NOT EXISTS idx_prods_month ON prods(month);

	CREATE TABLE IF NOT EXISTS group_prods (
		group_id INTEGER NOT NULL,
		prod_id INTEGER NOT NULL,
		PRIMARY KEY (group_id, prod_id)
	);

	CREATE INDEX IF NOT EXISTS idx_group_prods_prod_id ON group_prods(prod_id);
	`

	_, err := db.db.Exec(schema)
	return err
}

func (db *PouetDatabase) BuildIndex() error {
	if _, err := db.db.Exec("CREATE VIRTUAL TABLE groups_fts USING fts5(name, id)"); err != nil {
		return fmt.Errorf("create FTS index for groups: %w", err)
	}
	if _, err := db.db.Exec("INSERT INTO groups_fts (name, id) SELECT name, id FROM groups"); err != nil {
		return fmt.Errorf("populate FTS index for groups: %w", err)
	}

	if _, err := db.db.Exec("CREATE VIRTUAL TABLE prods_fts USING fts5(name, id)"); err != nil {
		return fmt.Errorf("create FTS index for prods: %w", err)
	}
	if _, err := db.db.Exec("INSERT INTO prods_fts (name, id) SELECT name, id FROM prods"); err != nil {
		return fmt.Errorf("populate FTS index for prods: %w", err)
	}

	return nil
}

func readJsonGz(filename string) (map[string]any, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open file %s: %w", filename, err)
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("unpack file %s: %w", filename, err)
	}
	defer gz.Close()

	var value map[string]any
	err = json.NewDecoder(gz).Decode(&value)
	if err != nil {
		return nil, fmt.Errorf("decode json from file %s: %w", filename, err)
	}

	return value, nil
}

func parsePouetDate(dateString string) (int, int, error) {
	if len(dateString) < 10 {
		return 0, 0, fmt.Errorf("date \"%s\" is invalid: expected YYYY-MM-DD format", dateString)
	}

	date, err := time.Parse("2006-01-02", dateString)
	if err == nil {
		return int(date.Year()), int(date.Month()), nil
	}

	date, err = time.Parse("2006", dateString[:4])
	if err != nil {
		return 0, 0, fmt.Errorf("parse date \"%s\": %w", dateString, err)
	}

	return int(date.Year()), 0, nil
}

func (db *PouetDatabase) ImportPouet(prodsfile string, groupsfile string) error {
	if prodsfile == "" || groupsfile == "" {
		return fmt.Errorf("when creating a new db, pouet data dumps are needed")
	}

	if err := db.createPouetTables(); err != nil {
		return fmt.Errorf("create tables: %w", err)
	}

	log.Printf("Importing prods...")
	if err := db.importProds(prodsfile); err != nil {
		return err
	}

	log.Printf("Importing groups...")
	if err := db.importGroups(groupsfile); err != nil {
		return err
	}

	log.Printf("Import done.")
	return nil
}

func (db *PouetDatabase) importProds(prodsfile string) error {
	prods, err := readJsonGz(prodsfile)
	if err != nil {
		return fmt.Errorf("unable to read prods from file %s: %w", prodsfile, err)
	}

	log.Printf("Loaded prods json into memory...")

	prodsArray, ok := prods["prods"].([]any)
	if !ok {
		return fmt.Errorf("prods field is not an array")
	}
	numProds := len(prodsArray)

	createdGroups := make(map[uint]bool)
	groupCount := 0

	tx, err := db.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	prodStmt, err := tx.Prepare(`INSERT INTO prods (id, name, year, month, video, rank, voteup, votepig, votedown, demozoo, screenshot)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare prod insert: %w", err)
	}
	defer prodStmt.Close()

	groupStmt, err := tx.Prepare(`INSERT INTO groups (id, name, disambiguation) VALUES (?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare group insert: %w", err)
	}
	defer groupStmt.Close()

	assocStmt, err := tx.Prepare(`INSERT OR IGNORE INTO group_prods (group_id, prod_id) VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare assoc insert: %w", err)
	}
	defer assocStmt.Close()

	for i, jprod := range prodsArray {
		prod, ok := jprod.(map[string]any)
		if !ok {
			log.Printf("Prod %d: invalid prod format", i)
			continue
		}

		pidStr, ok := prod["id"].(string)
		if !ok {
			log.Printf("Prod %d: missing id", i)
			continue
		}
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			log.Printf("Prod %d:%s: cannot parse id '%s': %v", i, prod["name"], pidStr, err)
			continue
		}

		name, ok := prod["name"].(string)
		if !ok {
			log.Printf("Prod %d:%d: missing name", i, pid)
			continue
		}
		jdate, found := prod["releaseDate"]

		var year, month int

		if found && jdate != nil {
			dateStr, ok := jdate.(string)
			if !ok {
				log.Printf("Prod %d:%s: invalid date format", i, name)
				continue
			}
			year, month, err = parsePouetDate(dateStr)
			if err != nil {
				log.Printf("Prod %d:%s: cannot parse '%+v' as date: %+v", pid, name, dateStr, err)
				continue
			}
		} else {
			log.Printf("Prod %v:%v has no date", pid, name)
		}

		rankStr, _ := prod["rank"].(string)
		rank, _ := strconv.Atoi(rankStr)
		voteUpStr, _ := prod["voteup"].(string)
		voteUp, _ := strconv.Atoi(voteUpStr)
		votePigStr, _ := prod["votepig"].(string)
		votePig, _ := strconv.Atoi(votePigStr)
		voteDownStr, _ := prod["votedown"].(string)
		voteDown, _ := strconv.Atoi(voteDownStr)

		var demozoo int
		if jsonDemozoo, haveDemozoo := prod["demozoo"]; haveDemozoo && jsonDemozoo != nil {
			demozooStr, _ := jsonDemozoo.(string)
			demozoo, _ = strconv.Atoi(demozooStr)
		}

		var video string
		if dlinks, have := prod["downloadLinks"]; have {
			array, ok := dlinks.([]any)
			if ok {
				for _, jlink := range array {
					link, ok := jlink.(map[string]any)
					if !ok {
						continue
					}
					linkType, ok := link["type"].(string)
					if !ok {
						continue
					}
					if strings.Contains(strings.ToLower(linkType), "youtube") {
						video, _ = link["link"].(string)
						break
					}
					if strings.Contains(strings.ToLower(linkType), "vimeo") {
						video, _ = link["link"].(string)
						break
					}
				}
			}
		}

		var screenshot string
		if shot, found := prod["screenshot"]; found && shot != nil {
			screenshot, _ = shot.(string)
		}

		_, err = prodStmt.Exec(pid, name, year, month, video, rank, voteUp, votePig, voteDown, demozoo, screenshot)
		if err != nil {
			return fmt.Errorf("create prod %d:%s: %w", pid, name, err)
		}

		jgroups, ok := prod["groups"].([]any)
		if !ok {
			log.Printf("Prod %d:%s: missing groups", i, name)
			continue
		}
		for _, jgroup := range jgroups {
			group, ok := jgroup.(map[string]any)
			if !ok {
				continue
			}
			gidStr, ok := group["id"].(string)
			if !ok {
				continue
			}
			gid, err := strconv.Atoi(gidStr)
			if err != nil {
				log.Printf("Cannot parse '%+v' as id: %+v", group["id"], err)
				continue
			}

			if !createdGroups[uint(gid)] {
				gname, _ := group["name"].(string)
				disambiguation, _ := group["disambiguation"].(string)
				_, err = groupStmt.Exec(uint(gid), gname, disambiguation)
				if err != nil {
					return fmt.Errorf("create group %d:%s: %w", gid, gname, err)
				}
				createdGroups[uint(gid)] = true
				groupCount++
			}

			_, err = assocStmt.Exec(uint(gid), pid)
			if err != nil {
				return fmt.Errorf("associate group %d with prod %d: %w", gid, pid, err)
			}
		}

		if (i+1)%1000 == 0 {
			log.Printf("Processed %d / %d", i+1, numProds)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit prods: %w", err)
	}

	log.Printf("Imported %d groups from prods", groupCount)
	return nil
}

func (db *PouetDatabase) importGroups(groupsfile string) error {
	groups, err := readJsonGz(groupsfile)
	if err != nil {
		return fmt.Errorf("unable to read groups from file %s: %w", groupsfile, err)
	}

	log.Printf("Loaded groups json into memory...")

	groupsArray, ok := groups["groups"].([]any)
	if !ok {
		return fmt.Errorf("groups field is not an array")
	}
	numGroups := len(groupsArray)

	updatedCount := 0
	createdCount := 0

	tx, err := db.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	updateStmt, err := tx.Prepare(`UPDATE groups SET name = ?, disambiguation = ? WHERE id = ?`)
	if err != nil {
		return fmt.Errorf("prepare group update: %w", err)
	}
	defer updateStmt.Close()

	insertStmt, err := tx.Prepare(`INSERT INTO groups (id, name, disambiguation) VALUES (?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare group insert: %w", err)
	}
	defer insertStmt.Close()

	for i, jgroup := range groupsArray {
		group, ok := jgroup.(map[string]any)
		if !ok {
			log.Printf("Group %d: invalid group format", i)
			continue
		}
		gidStr, ok := group["id"].(string)
		if !ok {
			log.Printf("Group %d: missing id", i)
			continue
		}
		gid, err := strconv.Atoi(gidStr)
		if err != nil {
			log.Printf("Group %d: cannot parse '%+v' as id: %+v", i, group["id"], err)
			continue
		}

		name, _ := group["name"].(string)
		disambiguation, _ := group["disambiguation"].(string)

		var exists bool
		err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM groups WHERE id = ?)", uint(gid)).Scan(&exists)
		if err != nil {
			log.Printf("Error checking group %d: %v", gid, err)
			continue
		}

		if !exists {
			_, err = insertStmt.Exec(uint(gid), name, disambiguation)
			if err != nil {
				return fmt.Errorf("create group %d:%s: %w", gid, name, err)
			}
			createdCount++
		} else {
			result, err := updateStmt.Exec(name, disambiguation, uint(gid))
			if err != nil {
				return fmt.Errorf("update group %d:%s: %w", gid, name, err)
			}
			rows, _ := result.RowsAffected()
			if rows == 0 {
				createdCount++
			} else {
				updatedCount++
			}
		}

		if (i+1)%1000 == 0 {
			log.Printf("Processed %d / %d groups", i+1, numGroups)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit groups: %w", err)
	}

	log.Printf("Imported %d new groups, updated %d existing groups", createdCount, updatedCount)
	return nil
}

func (db *PouetDatabase) FindGroups(name string) ([]Group, error) {
	const limit = 10

	groups := make([]Group, 0, limit)

	rows, err := db.db.Query(`
		SELECT g.id, g.name, g.disambiguation
		FROM groups g
		INNER JOIN groups_fts f ON f.id = g.id
		WHERE f.match ?
		ORDER BY f.rank
		LIMIT ?`, name, limit)

	ftsSuccess := false
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var g Group
			if err := rows.Scan(&g.ID, &g.Name, &g.Disambiguation); err != nil {
				return nil, fmt.Errorf("scan group: %w", err)
			}
			groups = append(groups, g)
		}
		ftsSuccess = true
	}

	if !ftsSuccess || len(groups) == 0 {
		rows2, err := db.db.Query(`
			SELECT id, name, disambiguation FROM groups
			WHERE name LIKE ?
			ORDER BY id
			LIMIT ?`, "%"+name+"%", limit)
		if err != nil {
			return nil, fmt.Errorf("search groups: %w", err)
		}
		defer rows2.Close()
		for rows2.Next() {
			var g Group
			if err := rows2.Scan(&g.ID, &g.Name, &g.Disambiguation); err != nil {
				continue
			}
			found := false
			for _, existing := range groups {
				if existing.ID == g.ID {
					found = true
					break
				}
			}
			if !found {
				groups = append(groups, g)
			}
		}
	}

	for i := range groups {
		g := &groups[i]
		var count int64
		if err := db.db.QueryRow("SELECT COUNT(*) FROM prods p INNER JOIN group_prods gp ON gp.prod_id = p.id WHERE gp.group_id = ?", g.ID).Scan(&count); err != nil {
			log.Printf("getCounts count prods for group %d: %v", g.ID, err)
		} else {
			g.ProdsCount = count
		}
		if err := db.db.QueryRow("SELECT COUNT(*) FROM greets WHERE greetee_id = ?", g.ID).Scan(&count); err != nil {
			log.Printf("getCounts count greets for group %d: %v", g.ID, err)
		} else {
			g.GreetsCount = count
		}
	}

	return groups, nil
}

func (db *PouetDatabase) FindProds(name string) ([]Prod, error) {
	const limit = 10

	prods := make([]Prod, 0, limit)

	rows, err := db.db.Query(`
		SELECT p.id, p.name, p.year, p.month, p.video, p.rank, p.voteup, p.votepig, p.votedown, p.demozoo, p.screenshot,
			   gp.group_id
		FROM prods p
		INNER JOIN prods_fts f ON f.id = p.id
		INNER JOIN group_prods gp ON gp.prod_id = p.id
		WHERE f.match ?
		ORDER BY f.rank`, name)

	ftsSuccess := false
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var p Prod
			var groupID uint
			if err := rows.Scan(&p.ID, &p.Name, &p.Year, &p.Month, &p.Video, &p.Rank, &p.VoteUp, &p.VotePig, &p.VoteDown, &p.Demozoo, &p.Screenshot, &groupID); err != nil {
				return nil, fmt.Errorf("scan prod: %w", err)
			}
			p.Groups = append(p.Groups, Group{ID: groupID})
			prods = append(prods, p)
		}
		ftsSuccess = true
	}

	if !ftsSuccess || len(prods) == 0 {
		rows2, err := db.db.Query(`
			SELECT p.id, p.name, p.year, p.month, p.video, p.rank, p.voteup, p.votepig, p.votedown, p.demozoo, p.screenshot,
				   gp.group_id
			FROM prods p
			INNER JOIN group_prods gp ON gp.prod_id = p.id
			WHERE p.name LIKE ?
			ORDER BY p.id
			LIMIT ?`, "%"+name+"%", limit)
		if err != nil {
			return nil, fmt.Errorf("search prods: %w", err)
		}
		defer rows2.Close()
		for rows2.Next() {
			var p Prod
			var groupID uint
			if err := rows2.Scan(&p.ID, &p.Name, &p.Year, &p.Month, &p.Video, &p.Rank, &p.VoteUp, &p.VotePig, &p.VoteDown, &p.Demozoo, &p.Screenshot, &groupID); err != nil {
				continue
			}
			found := false
			for _, existing := range prods {
				if existing.ID == p.ID {
					found = true
					break
				}
			}
			if !found {
				p.Groups = append(p.Groups, Group{ID: groupID})
				prods = append(prods, p)
			}
		}
	}

	return prods, nil
}

func (g *Group) getCounts(db *Database) error {
	var count int64
	if err := db.pouetDB.db.QueryRow("SELECT COUNT(*) FROM prods p INNER JOIN group_prods gp ON gp.prod_id = p.id WHERE gp.group_id = ?", g.ID).Scan(&count); err != nil {
		return fmt.Errorf("count prods for group %d: %w", g.ID, err)
	}
	g.ProdsCount = count

	if err := db.db.QueryRow("SELECT COUNT(*) FROM greets WHERE greetee_id = ?", g.ID).Scan(&count); err != nil {
		return fmt.Errorf("count greets for group %d: %w", g.ID, err)
	}
	g.GreetsCount = count

	return nil
}

func (db *PouetDatabase) GetProd(pid uint) (Prod, error) {
	var prod Prod
	row := db.db.QueryRow(`
		SELECT p.id, p.name, p.year, p.month, p.video, p.rank, p.voteup, p.votepig, p.votedown, p.demozoo, p.screenshot
		FROM prods p WHERE p.id = ?`, pid)

	if err := row.Scan(&prod.ID, &prod.Name, &prod.Year, &prod.Month, &prod.Video, &prod.Rank, &prod.VoteUp, &prod.VotePig, &prod.VoteDown, &prod.Demozoo, &prod.Screenshot); err != nil {
		if err == sql.ErrNoRows {
			return prod, fmt.Errorf("not found")
		}
		return prod, fmt.Errorf("unknown db error: %w", err)
	}

	rows, err := db.db.Query(`
		SELECT g.id, g.name, g.disambiguation
		FROM groups g
		INNER JOIN group_prods gp ON gp.group_id = g.id
		WHERE gp.prod_id = ?`, pid)
	if err != nil {
		return prod, fmt.Errorf("load groups for prod %d: %w", pid, err)
	}
	defer rows.Close()

	for rows.Next() {
		var group Group
		if err := rows.Scan(&group.ID, &group.Name, &group.Disambiguation); err != nil {
			return prod, fmt.Errorf("scan group: %w", err)
		}
		prod.Groups = append(prod.Groups, group)
	}

	return prod, nil
}

func (db *PouetDatabase) GetGroup(groupID uint) (Group, error) {
	var group Group
	row := db.db.QueryRow(`SELECT id, name, disambiguation FROM groups WHERE id = ?`, groupID)

	if err := row.Scan(&group.ID, &group.Name, &group.Disambiguation); err != nil {
		if err == sql.ErrNoRows {
			return group, fmt.Errorf("not found")
		}
		return group, fmt.Errorf("find group id=%d: %w", groupID, err)
	}

	return group, nil
}

func (db *PouetDatabase) Close() error {
	if db.db != nil {
		return db.db.Close()
	}
	return nil
}
