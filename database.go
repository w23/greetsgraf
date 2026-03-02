package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Greet struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
	UserID    uint
	Reference string
	ProdID    uint
	GreeteeID uint
}

type ProdGreet struct {
	ID          uint
	GreeteeID   uint
	GreeteeName string
	Reference   string
}

type GroupGreet struct {
	Prod      Prod
	Reference string
}

type Database struct {
	db      *sql.DB
	pouetDB *PouetDatabase
}

func DatabaseOpen(datafile, pouetDBFile string) (Database, error) {
	db, err := sql.Open("sqlite3", datafile)
	if err != nil {
		return Database{}, fmt.Errorf("open database file %s: %w", datafile, err)
	}

	pouetDB, err := PouetDatabaseOpen(pouetDBFile)
	if err != nil {
		return Database{}, fmt.Errorf("open pouet database file %s: %w", pouetDBFile, err)
	}

	return Database{db: db, pouetDB: &pouetDB}, nil
}

func (db *Database) createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS greets (
		id INTEGER PRIMARY KEY,
		created_at TIMESTAMP,
		updated_at TIMESTAMP,
		deleted_at TIMESTAMP,
		user_id INTEGER,
		reference TEXT,
		prod_id INTEGER NOT NULL,
		greetee_id INTEGER NOT NULL,
		UNIQUE (prod_id, greetee_id)
	);

	CREATE INDEX IF NOT EXISTS idx_greets_prod_id ON greets(prod_id);
	CREATE INDEX IF NOT EXISTS idx_greets_greetee_id ON greets(greetee_id);
	`

	_, err := db.db.Exec(schema)
	return err
}

func (db *Database) BuildIndex() error {
	// FTS5 indexes are now in pouet database
	return db.pouetDB.BuildIndex()
}

func (db *Database) ImportPouet(prodsfile string, groupsfile string) error {
	if prodsfile == "" || groupsfile == "" {
		return fmt.Errorf("when creating a new db, pouet data dumps are needed")
	}

	if err := db.createTables(); err != nil {
		return fmt.Errorf("create tables: %w", err)
	}

	log.Printf("Importing prods...")
	if err := db.pouetDB.ImportPouet(prodsfile, groupsfile); err != nil {
		return err
	}

	log.Printf("Import done.")
	return nil
}

func (db *Database) FindGroups(name string) ([]Group, error) {
	return db.pouetDB.FindGroups(name)
}

func (db *Database) FindProds(name string) ([]Prod, error) {
	return db.pouetDB.FindProds(name)
}

func (db *Database) GetProd(pid uint) (Prod, error) {
	return db.pouetDB.GetProd(pid)
}

func (db *Database) GetGroup(groupID uint) (Group, error) {
	return db.pouetDB.GetGroup(groupID)
}

func (db *Database) GetProdGreets(prodID uint) ([]ProdGreet, error) {
	var greets []ProdGreet
	rows, err := db.db.Query(`
		SELECT gr.id, gr.greetee_id, gr.reference
		FROM greets gr
		WHERE gr.prod_id = ?`, prodID)

	if err != nil {
		return nil, fmt.Errorf("get greets for prod=%d: %w", prodID, err)
	}
	defer rows.Close()

	for rows.Next() {
		var greetID, greeteeID uint
		var reference string
		if err := rows.Scan(&greetID, &greeteeID, &reference); err != nil {
			return nil, fmt.Errorf("scan greet: %w", err)
		}

		group, err := db.pouetDB.GetGroup(greeteeID)
		if err != nil {
			log.Printf("GetGroup for greetee_id=%d: %v", greeteeID, err)
			continue
		}

		greets = append(greets, ProdGreet{
			ID:          greetID,
			GreeteeID:   greeteeID,
			GreeteeName: group.Name,
			Reference:   reference,
		})
	}

	return greets, nil
}

func (db *Database) GetGroupGreets(groupID uint) ([]GroupGreet, error) {
	rows, err := db.db.Query(`
		SELECT gr.id, gr.prod_id, gr.reference
		FROM greets gr
		WHERE gr.greetee_id = ?`, groupID)
	if err != nil {
		return nil, fmt.Errorf("get greets for greetee_id=%d: %w", groupID, err)
	}
	defer rows.Close()
	var greets = make([]GroupGreet, 0, 4)
	for rows.Next() {
		var prodID, greetID uint
		var reference string

		if err := rows.Scan(&greetID, &prodID, &reference); err != nil {
			log.Printf("GetGroupGreets: scan greet error: %v", err)
			continue
		}

		prod, err := db.pouetDB.GetProd(prodID)
		if err != nil {
			log.Printf("GetGroupGreets: get prod %d: %v", prodID, err)
			continue
		}

		for i := range prod.Groups {
			g := &prod.Groups[i]
			var count int64
			if err := db.pouetDB.db.QueryRow("SELECT COUNT(*) FROM prods p INNER JOIN group_prods gp ON gp.prod_id = p.id WHERE gp.group_id = ?", g.ID).Scan(&count); err != nil {
				log.Printf("GetGroupGreets: count prods for group %d: %v", g.ID, err)
			} else {
				g.ProdsCount = count
			}
			if err := db.db.QueryRow("SELECT COUNT(*) FROM greets WHERE greetee_id = ?", g.ID).Scan(&count); err != nil {
				log.Printf("GetGroupGreets: count greets for group %d: %v", g.ID, err)
			} else {
				g.GreetsCount = count
			}
		}

		greets = append(greets, GroupGreet{
			Prod:      prod,
			Reference: reference,
		})
	}

	return greets, nil
}

func (db *Database) Greet(prodID uint, groupID uint, note string) (uint, error) {
	tx, err := db.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	prodExists, err := db.pouetDB.ProdExists(prodID)
	if err != nil {
		return 0, err
	}
	if !prodExists {
		return 0, fmt.Errorf("prod not found: id=%d", prodID)
	}

	groupExists, err := db.pouetDB.GroupExists(groupID)
	if err != nil {
		return 0, err
	}
	if !groupExists {
		return 0, fmt.Errorf("group not found: id=%d", groupID)
	}

	if len(note) > 500 {
		return 0, fmt.Errorf("note too long: max 500 characters, got %d", len(note))
	}

	log.Printf("Creating greet: prod_id=%d, group_id=%d, note=%s", prodID, groupID, note)

	result, err := tx.Exec(`
		INSERT INTO greets (reference, greetee_id, prod_id)
		VALUES (?, ?, ?)`, note, groupID, prodID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return 0, fmt.Errorf("duplicate greet: prod_id=%d, greetee_id=%d", prodID, groupID)
		}
		return 0, fmt.Errorf("create greet: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last insert id: %w", err)
	}
	log.Printf("Greet created with ID: %d", id)

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("tx commit: %w", err)
	}

	return uint(id), nil
}

func (db *Database) DeleteGreet(greetID uint) (bool, error) {
	result, err := db.db.Exec("DELETE FROM greets WHERE id = ?", greetID)
	if err != nil {
		return false, fmt.Errorf("delete greet=%d: %w", greetID, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("get rows affected for greet=%d: %w", greetID, err)
	}
	if rows == 0 {
		return false, nil
	}

	return true, nil
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
	if err := db.db.QueryRow("SELECT COUNT(*) FROM greets").Scan(&stats.TotalGreets); err != nil {
		log.Printf("count greets: %v", err)
	}
	if count, err := db.pouetDB.CountProds(); err != nil {
		log.Printf("count prods: %v", err)
	} else {
		stats.TotalProds = count
	}
	if count, err := db.pouetDB.CountGroups(); err != nil {
		log.Printf("count groups: %v", err)
	} else {
		stats.TotalGroups = count
	}
	if err := db.db.QueryRow("SELECT COUNT(DISTINCT prod_id) FROM greets").Scan(&stats.ProdsWithGreets); err != nil {
		log.Printf("count prods with greets: %v", err)
	}
	if err := db.db.QueryRow("SELECT COUNT(DISTINCT greetee_id) FROM greets").Scan(&stats.GreetedGroups); err != nil {
		log.Printf("count greeted groups: %v", err)
	}
	return stats
}

func (db *Database) GetMostGreetedGroups(limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 1000 {
		limit = 1000
	}

	rows, err := db.db.Query(`
		SELECT greetee_id, COUNT(*) as count
		FROM greets
		GROUP BY greetee_id
		ORDER BY count DESC
		LIMIT ?`, limit)

	if err != nil {
		return nil, fmt.Errorf("get most %d greeted groups: %w", limit, err)
	}
	defer rows.Close()

	var results = make([]map[string]any, 0, limit)
	for rows.Next() {
		var groupID uint
		var count int64

		if err := rows.Scan(&groupID, &count); err != nil {
			return []map[string]any{}, fmt.Errorf("scan result: %w", err)
		}

		group, err := db.pouetDB.GetGroup(groupID)
		if err != nil {
			log.Printf("GetGroup for group_id=%d: %v", groupID, err)
			continue
		}

		var result = make(map[string]any)
		result["group_id"] = group.ID
		result["group_name"] = group.Name
		result["count"] = count
		results = append(results, result)
	}

	return results, nil
}

func (db *Database) Close() error {
	if db.db != nil {
		db.db.Close()
	}
	if db.pouetDB != nil {
		db.pouetDB.Close()
	}
	return nil
}
