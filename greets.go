package main

import (
	"fmt"
	"net/http"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Greet struct {
	gorm.Model
	UserID    uint `gorm:"index"`
	Reference string
	ProdID    uint `gorm:"uniqueIndex:greets_prod_group"`
	GreeteeID uint `gorm:"uniqueIndex:greets_prod_group"`
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

type DatabaseStats struct {
	TotalGreets     int64
	TotalProds      int64
	TotalGroups     int64
	ProdsWithGreets int64
	GreetedGroups   int64
}

type Greets struct {
	db      *gorm.DB
	pouetDB *Pouet
}

func GreetsOpen(datafile string, pouetDB *Pouet) (Greets, error) {
	db, err := gorm.Open(sqlite.Open(datafile), &gorm.Config{})

	if err != nil {
		return Greets{nil, nil}, fmt.Errorf("open greets database file %s: %w", datafile, err)
	}

	return Greets{db, pouetDB}, err
}

func (db *Greets) AutoMigrate() {
	db.db.AutoMigrate(&Greet{})
}

func (db *Greets) Greet(prodID uint, groupID uint, note string) (uint, error) {
	var prod Prod
	if err := db.db.First(&prod, "id = ?", prodID).Error; err != nil {
		return 0, fmt.Errorf("prod %d not found: %w", prodID, err)
	}

	var group Group
	if err := db.db.First(&group, "id = ?", groupID).Error; err != nil {
		return 0, fmt.Errorf("group %d not found: %w", groupID, err)
	}

	greet := Greet{
		ProdID:    prodID,
		GreeteeID: groupID,
		Reference: note,
	}

	if err := db.db.Create(&greet).Error; err != nil {
		return 0, fmt.Errorf("create greet: %w", err)
	}

	return greet.ID, nil
}

func (db *Greets) DeleteGreet(greetID uint) (bool, error) {
	query := db.db.Unscoped().Delete(&Greet{}, "id = ?", greetID)
	if query.Error == gorm.ErrRecordNotFound {
		return false, nil
	} else if query.Error != nil {
		return false, fmt.Errorf("delete greet=%d: %w", greetID, query.Error)
	}

	if query.RowsAffected == 0 {
		return false, nil
	}

	return true, nil
}

func (db *Greets) GetProdGreets(r *http.Request, prodID any) ([]ProdGreet, error) {
	var greets []ProdGreet
	query := r.Context().Value("writableDB").(*Greets).db.Table("greets").
		Select("greets.greetee_id as GreeteeID, groups.name as GreeteeName, greets.reference as Reference").
		Where("greets.prod_id = ?", prodID).
		Joins("INNER JOIN groups ON groups.id = greets.greetee_id").
		Find(&greets)

	if query.Error != nil {
		return []ProdGreet{}, fmt.Errorf("get greets for prod=%v: %w", prodID, query.Error)
	}

	return greets, nil
}

func (db *Greets) GetGroupGreets(r *http.Request, groupID any) ([]GroupGreet, error) {
	var raw_greets []Greet
	query := r.Context().Value("writableDB").(*Greets).db.Find(&raw_greets, "greetee_id = ?", groupID)

	if query.Error != nil {
		return []GroupGreet{}, fmt.Errorf("get greets for greetee_id=%v: %w", groupID, query.Error)
	}

	var greets []GroupGreet

	for i := range raw_greets {
		raw_greet := &raw_greets[i]
		var prod Prod
		db.pouetDB.db.Find(&prod, "id = ?", raw_greet.ProdID).Association("Groups")
		db.pouetDB.db.Model(&prod).Association("Groups").Find(&prod.Groups)
		for j := range prod.Groups {
			prod.Groups[j].getCounts(db.pouetDB.db)
		}
		greets = append(greets, GroupGreet{
			Prod:      prod,
			Reference: raw_greet.Reference,
		})
	}

	return greets, nil
}

func (db *Greets) GetMostGreetedGroups(r *http.Request, limit int) ([]map[string]any, error) {
	var results []map[string]any
	query := r.Context().Value("writableDB").(*Greets).db.Model(Greet{}).Select("greets.greetee_id AS group_id, groups.name AS group_name, COUNT(DISTINCT greets.id) AS count").Joins("INNER JOIN groups ON groups.id = greets.greetee_id").Group("greets.greetee_id").Order("count DESC").Limit(limit).Find(&results)

	if query.Error != nil {
		return []map[string]any{}, fmt.Errorf("get most %d greeted groups: %w", limit, query.Error)
	}

	return results, nil
}

func (db *Greets) GetStats(r *http.Request) DatabaseStats {
	var stats DatabaseStats
	writableDB := r.Context().Value("writableDB").(*Greets)
	writableDB.db.Model(Greet{}).Count(&stats.TotalGreets)
	db.pouetDB.db.Model(Prod{}).Count(&stats.TotalProds)
	db.pouetDB.db.Model(Group{}).Count(&stats.TotalGroups)
	writableDB.db.Model(Greet{}).Distinct("prod_id").Count(&stats.ProdsWithGreets)
	writableDB.db.Model(Greet{}).Distinct("greetee_id").Count(&stats.GreetedGroups)
	return stats
}
