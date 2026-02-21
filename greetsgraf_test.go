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

	t.Run("GroupSearchTheBlackLotus", func(t *testing.T) {
		groupResp1, err := http.Get(server.URL + "/v1/groups/search?name=The%20Black%20Lotus")
		require.NoError(t, err)
		defer groupResp1.Body.Close()

		assert.Equal(t, http.StatusOK, groupResp1.StatusCode)

		bodyBytes, err = io.ReadAll(groupResp1.Body)
		require.NoError(t, err)
		t.Logf("Group search response for 'The Black Lotus': %s", string(bodyBytes))

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
	})

	t.Run("GroupSearchExceed", func(t *testing.T) {
		groupResp2, err := http.Get(server.URL + "/v1/groups/search?name=Exceed")
		require.NoError(t, err)
		defer groupResp2.Body.Close()

		assert.Equal(t, http.StatusOK, groupResp2.StatusCode)

		bodyBytes, err = io.ReadAll(groupResp2.Body)
		require.NoError(t, err)
		t.Logf("Group search response for 'Exceed': %s", string(bodyBytes))

		var groups2 []GroupSearchResponse
		err = json.Unmarshal(bodyBytes, &groups2)
		require.NoError(t, err)

		assert.Len(t, groups2, 1)
		assert.Equal(t, groups2[0], GroupSearchResponse{
			ID:             uint(2),
			Name:           "Exceed",
			Disambiguation: "pc/c64/c16",
			ProdsCount:     24,
			GreetsCount:    0,
		})
	})

	t.Run("ProdGetID1", func(t *testing.T) {
		prodResp, err := http.Get(server.URL + "/v1/prods/1")
		require.NoError(t, err)
		defer prodResp.Body.Close()

		bodyBytes, err = io.ReadAll(prodResp.Body)
		require.NoError(t, err)
		t.Logf("Prod get response for ID 1: %s", string(bodyBytes))

		var prodRespBody ProdGetResponse
		err = json.Unmarshal(bodyBytes, &prodRespBody)
		require.NoError(t, err)

		assert.Equal(t, prodRespBody, ProdGetResponse{
			ID:         uint(1),
			Name:       "Astral Blur",
			Year:       1997,
			Month:      3,
			Day:        15,
			Video:      "https://www.youtube.com/watch?v=eZyLSHyUGBY",
			Rank:       712,
			VoteUp:     84,
			VotePig:    18,
			VoteDown:   5,
			Demozoo:    11,
			Screenshot: "http://content.pouet.net/files/screenshots/00000/00000001.jpg",
			Groups: []ResponseGroup{
				{ID: 1, Name: "The Black Lotus", Disambiguation: ""},
			},
			Greets: nil,
		})
	})

	t.Run("ProdGetID2", func(t *testing.T) {
		prodResp2, err := http.Get(server.URL + "/v1/prods/2")
		require.NoError(t, err)
		defer prodResp2.Body.Close()

		bodyBytes, err = io.ReadAll(prodResp2.Body)
		require.NoError(t, err)
		t.Logf("Prod get response for ID 2: %s", string(bodyBytes))

		var prodRespBody2 ProdGetResponse
		err = json.Unmarshal(bodyBytes, &prodRespBody2)
		require.NoError(t, err)

		assert.Equal(t, prodRespBody2, ProdGetResponse{
			ID:         uint(2),
			Name:       "Jizz",
			Year:       1997,
			Month:      7,
			Day:        15,
			Video:      "https://www.youtube.com/watch?v=iXgseVYvhek",
			Rank:       364,
			VoteUp:     104,
			VotePig:    11,
			VoteDown:   2,
			Demozoo:    12,
			Screenshot: "http://content.pouet.net/files/screenshots/00000/00000002.jpg",
			Groups: []ResponseGroup{
				{ID: 1, Name: "The Black Lotus", Disambiguation: ""},
			},
			Greets: nil,
		})
	})
}
