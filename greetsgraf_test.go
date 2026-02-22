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

func makeRequest[T any](t *testing.T, url string, expectedStatus int) T {
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, expectedStatus, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result T
	err = json.Unmarshal(bodyBytes, &result)
	require.NoError(t, err)

	return result
}

type StatsResponse struct {
	TotalGreets     int64 `json:"TotalGreets"`
	TotalProds      int64 `json:"TotalProds"`
	TotalGroups     int64 `json:"TotalGroups"`
	ProdsWithGreets int64 `json:"ProdsWithGreets"`
	GreetedGroups   int64 `json:"GreetedGroups"`
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
		ID    uint          `json:"id"`
		Group ResponseGroup `json:"group"`
		Note  string        `json:"note"`
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

	statsResponse := makeRequest[StatsResponse](t, server.URL+"/v1/stats", http.StatusOK)

	assert.Equal(t, statsResponse, StatsResponse{
		TotalGreets:     0,
		TotalProds:      420,
		TotalGroups:     64,
		ProdsWithGreets: 0,
		GreetedGroups:   0,
	})

	t.Run("GroupSearchTheBlackLotus", func(t *testing.T) {
		groups1 := makeRequest[[]GroupSearchResponse](t, server.URL+"/v1/groups/search?name=The%20Black%20Lotus", http.StatusOK)
		t.Logf("Group search response for 'The Black Lotus': %v", groups1)

		assert.Equal(t, groups1, []GroupSearchResponse{
			{
				ID:             uint(1),
				Name:           "The Black Lotus",
				Disambiguation: "",
				ProdsCount:     64,
				GreetsCount:    0,
			},
		})
	})

	t.Run("GroupSearchExceed", func(t *testing.T) {
		groups2 := makeRequest[[]GroupSearchResponse](t, server.URL+"/v1/groups/search?name=Exceed", http.StatusOK)
		t.Logf("Group search response for 'Exceed': %v", groups2)

		assert.Equal(t, groups2, []GroupSearchResponse{
			{
				ID:             uint(2),
				Name:           "Exceed",
				Disambiguation: "pc/c64/c16",
				ProdsCount:     24,
				GreetsCount:    0,
			},
		})
	})

	t.Run("GroupSearchNoResults", func(t *testing.T) {
		groups := makeRequest[[]GroupSearchResponse](t, server.URL+"/v1/groups/search?name=nonexistentgroup12345", http.StatusOK)
		t.Logf("Group search response for non-existent group: %v", groups)

		assert.Equal(t, groups, []GroupSearchResponse{})
	})

	t.Run("ProdGetID1", func(t *testing.T) {
		prodRespBody := makeRequest[ProdGetResponse](t, server.URL+"/v1/prods/1", http.StatusOK)
		t.Logf("Prod get response for ID 1: %v", prodRespBody)

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
		prodRespBody2 := makeRequest[ProdGetResponse](t, server.URL+"/v1/prods/2", http.StatusOK)
		t.Logf("Prod get response for ID 2: %v", prodRespBody2)

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

	t.Run("GroupSearchInvalidFTS", func(t *testing.T) {
		groups := makeRequest[[]GroupSearchResponse](t, server.URL+"/v1/groups/search?name=*invalid*", http.StatusOK)
		t.Logf("Group search with invalid FTS query: %v", groups)

		assert.Equal(t, groups, []GroupSearchResponse{})
	})

	t.Run("GroupSearchEmptyName", func(t *testing.T) {
		groups := makeRequest[[]GroupSearchResponse](t, server.URL+"/v1/groups/search?name=", http.StatusOK)
		t.Logf("Group search with empty name: %v", groups)

		assert.Equal(t, groups, []GroupSearchResponse{})
	})

	t.Run("GroupSearchAndromeda", func(t *testing.T) {
		groups := makeRequest[[]GroupSearchResponse](t, server.URL+"/v1/groups/search?name=Andromeda", http.StatusOK)
		t.Logf("Group search response for 'Andromeda': %v", groups)

		assert.Equal(t, groups, []GroupSearchResponse{
			{
				ID:             uint(196),
				Name:           "Andromeda",
				Disambiguation: "",
				ProdsCount:     27,
				GreetsCount:    0,
			},
			{
				ID:             uint(1317),
				Name:           "Andromeda Software Development",
				Disambiguation: "",
				ProdsCount:     58,
				GreetsCount:    0,
			},
		})
	})

	t.Run("GroupSearchBrausch", func(t *testing.T) {
		groups := makeRequest[[]GroupSearchResponse](t, server.URL+"/v1/groups/search?name=brausch", http.StatusOK)
		t.Logf("Group search response for 'brausch': %v", groups)

		assert.Equal(t, groups, []GroupSearchResponse{
			{
				ID:             uint(322),
				Name:           "Farbrausch",
				Disambiguation: "",
				ProdsCount:     148,
				GreetsCount:    0,
			},
		})
	})

	t.Run("GroupSearchSoftware", func(t *testing.T) {
		groups := makeRequest[[]GroupSearchResponse](t, server.URL+"/v1/groups/search?name=software", http.StatusOK)
		t.Logf("Group search response for 'software': %v", groups)

		assert.Equal(t, groups, []GroupSearchResponse{
			{
				ID:             uint(1317),
				Name:           "Andromeda Software Development",
				Disambiguation: "",
				ProdsCount:     58,
				GreetsCount:    0,
			},
		})
	})

	t.Run("GroupSearchTBC", func(t *testing.T) {
		groups := makeRequest[[]GroupSearchResponse](t, server.URL+"/v1/groups/search?name=TBC", http.StatusOK)
		t.Logf("Group search response for 'TBC': %v", groups)

		assert.Equal(t, groups, []GroupSearchResponse{
			{
				ID:             uint(1623),
				Name:           "TBC",
				Disambiguation: "",
				ProdsCount:     47,
				GreetsCount:    0,
			},
		})
	})

	t.Run("GroupSearchOrb", func(t *testing.T) {
		groups := makeRequest[[]GroupSearchResponse](t, server.URL+"/v1/groups/search?name=orb", http.StatusOK)
		t.Logf("Group search response for 'orb': %v", groups)

		assert.Equal(t, groups, []GroupSearchResponse{
			{
				ID:             uint(7439),
				Name:           "Orb",
				Disambiguation: "",
				ProdsCount:     17,
				GreetsCount:    0,
			},
		})
	})

	t.Run("GroupSearchSingleLetterA", func(t *testing.T) {
		groups := makeRequest[[]GroupSearchResponse](t, server.URL+"/v1/groups/search?name=a", http.StatusOK)
		t.Logf("Group search response for 'a': %v", groups)

		assert.Equal(t, groups, []GroupSearchResponse{
			{
				ID:             uint(1),
				Name:           "The Black Lotus",
				Disambiguation: "",
				ProdsCount:     64,
				GreetsCount:    0,
			},
			{
				ID:             uint(196),
				Name:           "Andromeda",
				Disambiguation: "",
				ProdsCount:     27,
				GreetsCount:    0,
			},
			{
				ID:             uint(322),
				Name:           "Farbrausch",
				Disambiguation: "",
				ProdsCount:     148,
				GreetsCount:    0,
			},
			{
				ID:             uint(697),
				Name:           "Rgba",
				Disambiguation: "",
				ProdsCount:     37,
				GreetsCount:    0,
			},
			{
				ID:             uint(1317),
				Name:           "Andromeda Software Development",
				Disambiguation: "",
				ProdsCount:     58,
				GreetsCount:    0,
			},
		})
	})
}
