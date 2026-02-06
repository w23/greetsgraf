package main

import (
	"fmt"

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

type Greets struct {
	db *gorm.DB
}

func GreetsOpen(datafile string) (Greets, error) {
	db, err := gorm.Open(sqlite.Open(datafile), &gorm.Config{})

	if err != nil {
		return Greets{nil}, fmt.Errorf("open greets database file %s: %w", datafile, err)
	}

	return Greets{db}, err
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
