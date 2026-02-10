package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testDB() *gorm.DB {
	dbfile := "/tmp/test_greetsgraf.db"
	os.Remove(dbfile)
	db, err := gorm.Open(sqlite.Open(dbfile), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	return db
}

func TestStatsEndpoint(t *testing.T) {
	db := testDB()
	db.AutoMigrate(&Group{}, &Prod{}, &Greet{})

	server := httptest.NewServer(Server(db))
	defer server.Close()

	resp, err := http.Get(server.URL + "/v1/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}
