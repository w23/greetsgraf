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

type GroupSearchResponse struct {
	ID             uint   `json:"id"`
	Name           string `json:"name"`
	Disambiguation string `json:"disambiguation"`
	ProdsCount     int64  `json:"prodsCount"`
	GreetsCount    int64  `json:"greetsCount"`
}

type ResponseGroup struct {
	ID             uint   `json:"id"`
	Name           string `json:"name"`
	Disambiguation string `json:"disambiguation"`
}

type ProdGetResponse struct {
	ID         uint            `json:"id"`
	Name       string          `json:"name"`
	Year       int             `json:"year"`
	Month      int             `json:"month"`
	Day        int             `json:"day"`
	Video      string          `json:"video"`
	Rank       int             `json:"rank"`
	VoteUp     int             `json:"voteup"`
	VotePig    int             `json:"votepig"`
	VoteDown   int             `json:"votedown"`
	Demozoo    int             `json:"demozoo"`
	Screenshot string          `json:"screenshot"`
	Groups     []ResponseGroup `json:"groups"`
	Greets     []struct {
		ID    uint
		Group ResponseGroup
		Note  string
	} `json:"greets"`
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

	t.Run("QueryTwoGroupsAndProds", func(t *testing.T) {
		groupResp1, err := http.Get(server.URL + "/v1/groups/search?name=The%20Black%20Lotus")
		require.NoError(t, err)
		defer groupResp1.Body.Close()

		assert.Equal(t, http.StatusOK, groupResp1.StatusCode)

		bodyBytes, err = io.ReadAll(groupResp1.Body)
		require.NoError(t, err)

		var groups1 []GroupSearchResponse
		err = json.Unmarshal(bodyBytes, &groups1)
		require.NoError(t, err)

		assert.Len(t, groups1, 1)
		assert.Equal(t, groups1[0], GroupSearchResponse{
			ID:             uint(1),
			Name:           "The Black Lotus",
			Disambiguation: "",
			ProdsCount:     64,
			GreetsCount:    0,
		})

		groupResp2, err := http.Get(server.URL + "/v1/groups/search?name=Exceed")
		require.NoError(t, err)
		defer groupResp2.Body.Close()

		assert.Equal(t, http.StatusOK, groupResp2.StatusCode)

		var groups2 []GroupSearchResponse
		bodyBytes, err = io.ReadAll(groupResp2.Body)
		require.NoError(t, err)
		err = json.Unmarshal(bodyBytes, &groups2)
		require.NoError(t, err)

		assert.Len(t, groups2, 1)
		assert.Equal(t, groups2[0].ID, uint(2))
		assert.Equal(t, groups2[0].Name, "Exceed")

		prodResp1, err := http.Get(server.URL + "/v1/prods/1")
		require.NoError(t, err)
		defer prodResp1.Body.Close()

		assert.Equal(t, http.StatusOK, prodResp1.StatusCode)

		var prod1 ProdGetResponse
		bodyBytes, err = io.ReadAll(prodResp1.Body)
		require.NoError(t, err)
		err = json.Unmarshal(bodyBytes, &prod1)
		require.NoError(t, err)

		assert.Equal(t, prod1.ID, uint(1))
		assert.Equal(t, prod1.Name, "Astral Blur")
		assert.Len(t, prod1.Groups, 1)
		assert.Equal(t, prod1.Groups[0].ID, uint(1))
		assert.Equal(t, prod1.Groups[0].Name, "The Black Lotus")

		prodResp2, err := http.Get(server.URL + "/v1/prods/2")
		require.NoError(t, err)
		defer prodResp2.Body.Close()

		assert.Equal(t, http.StatusOK, prodResp2.StatusCode)

		var prod2 ProdGetResponse
		bodyBytes, err = io.ReadAll(prodResp2.Body)
		require.NoError(t, err)
		err = json.Unmarshal(bodyBytes, &prod2)
		require.NoError(t, err)

		assert.Equal(t, prod2.ID, uint(2))
		assert.Equal(t, prod2.Name, "Jizz")
		assert.Len(t, prod2.Groups, 1)
		assert.Equal(t, prod2.Groups[0].ID, uint(1))
		assert.Equal(t, prod2.Groups[0].Name, "The Black Lotus")
	})

	prodResp, err := http.Get(server.URL + "/v1/prods/1")
	require.NoError(t, err)
	defer prodResp.Body.Close()

	var prodRespBody ProdGetResponse
	bodyBytes, err = io.ReadAll(prodResp.Body)
	require.NoError(t, err)
	err = json.Unmarshal(bodyBytes, &prodRespBody)
	require.NoError(t, err)

	assert.Equal(t, prodRespBody.ID, uint(1))
	assert.Equal(t, prodRespBody.Name, "Astral Blur")
}
