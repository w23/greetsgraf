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

func TestIngestAndRetrieve(t *testing.T) {
	db, err := SetupDatabase(SetupArgs{
		DBFile:      ":memory:?cache=shared",
		Create:      true,
		PouetProds:  "./test/pouet-prods.json.gz",
		PouetGroups: "./test/pouet-groups.json.gz",
		BuildIndex:  true,
	})
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(Server(db))
	defer server.Close()

	statsResp, err := http.Get(server.URL + "/v1/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer statsResp.Body.Close()

	if statsResp.StatusCode != http.StatusOK {
		t.Errorf("Expected stats status 200, got %d", statsResp.StatusCode)
	}

	prodResp, err := http.Get(server.URL + "/v1/prods/1")
	if err != nil {
		t.Fatal(err)
	}
	defer prodResp.Body.Close()

	if prodResp.StatusCode != http.StatusOK {
		t.Errorf("Expected prod status 200, got %d", prodResp.StatusCode)
	}
}
