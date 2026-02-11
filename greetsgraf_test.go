package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatsEndpoint(t *testing.T) {
	db, err := SetupDatabase(SetupArgs{
		DBFile: ":memory:?cache=shared",
		Create: false,
	})
	if err != nil {
		t.Fatal(err)
	}

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
