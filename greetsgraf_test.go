package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatsEndpoint(t *testing.T) {
	db, err := SetupDatabase(SetupArgs{
		DBFile: ":memory:?cache=shared",
		Create: false,
	})
	require.NoError(t, err)

	server := httptest.NewServer(Server(db))
	defer server.Close()

	resp, err := http.Get(server.URL + "/v1/stats")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestIngestAndRetrieve(t *testing.T) {
	db, err := SetupDatabase(SetupArgs{
		DBFile:      ":memory:?cache=shared",
		Create:      true,
		PouetProds:  "./test/pouet-prods.json.gz",
		PouetGroups: "./test/pouet-groups.json.gz",
		BuildIndex:  true,
	})
	require.NoError(t, err)

	server := httptest.NewServer(Server(db))
	defer server.Close()

	statsResp, err := http.Get(server.URL + "/v1/stats")
	require.NoError(t, err)
	defer statsResp.Body.Close()

	assert.Equal(t, http.StatusOK, statsResp.StatusCode)

	prodResp, err := http.Get(server.URL + "/v1/prods/1")
	require.NoError(t, err)
	defer prodResp.Body.Close()

	assert.Equal(t, http.StatusOK, prodResp.StatusCode)
}
