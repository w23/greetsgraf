package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type StatsResponse struct {
	TotalGreets     int `json:"TotalGreets"`
	TotalProds      int `json:"TotalProds"`
	TotalGroups     int `json:"TotalGroups"`
	ProdsWithGreets int `json:"ProdsWithGreets"`
	GreetedGroups   int `json:"GreetedGroups"`
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

	var bodyBytes []byte
	bodyBytes, err = io.ReadAll(statsResp.Body)
	require.NoError(t, err)

	var statsResponse StatsResponse
	err = json.Unmarshal(bodyBytes, &statsResponse)
	require.NoError(t, err)

	assert.Equal(t, statsResponse.TotalGreets, 0)
	assert.Equal(t, statsResponse.TotalProds, 420)
	assert.Equal(t, statsResponse.TotalGroups, 64)
	assert.Equal(t, statsResponse.ProdsWithGreets, 0)
	assert.Equal(t, statsResponse.GreetedGroups, 0)

	prodResp, err := http.Get(server.URL + "/v1/prods/1")
	require.NoError(t, err)
	defer prodResp.Body.Close()

	assert.Equal(t, http.StatusOK, prodResp.StatusCode)
}
