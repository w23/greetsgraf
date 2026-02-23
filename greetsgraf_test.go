package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ResponseGroup struct {
	ID             uint   `json:"id"`
	Name           string `json:"name"`
	Disambiguation string `json:"disambiguation"`
}

type ResponseGreet struct {
	ID    uint          `json:"ID"`
	Group ResponseGroup `json:"Group"`
	Note  string        `json:"Note"`
}

type GroupGreetsResponse struct {
	ID        uint   `json:"ID"`
	ProdID    uint   `json:"ProdID"`
	Prod      Prod   `json:"Prod"`
	Reference string `json:"Reference"`
}

type GroupGreetedResponse struct {
	GroupID   uint   `json:"group_id"`
	GroupName string `json:"group_name"`
	Count     int64  `json:"count"`
}

type ProdGreetsResponse struct {
	GreeteeID uint   `json:"GreeteeID"`
	GreeteeName string `json:"GreeteeName"`
	Reference string `json:"Reference"`
}

type GroupSearchResponse struct {
	ID             uint   `json:"id"`
	Name           string `json:"name"`
	Disambiguation string `json:"disambiguation"`
	ProdsCount     int64  `json:"prodsCount"`
	GreetsCount    int64  `json:"greetsCount"`
}

type ProdGetResponse struct {
	ID         uint            `json:"id"`
	Name       string          `json:"name"`
	Year       int             `json:"year"`
	Month      int             `json:"month"`
	Day        int             `json:"day"`
	Video      string          `json:"video"`
	Rank       int             `json:"rank"`
	VoteUp     int             `json:"voteUp"`
	VotePig    int             `json:"votePig"`
	VoteDown   int             `json:"voteDown"`
	Demozoo    int             `json:"demozoo"`
	Screenshot string          `json:"screenshot"`
	Groups     []ResponseGroup `json:"groups"`
	Greets     []ResponseGreet `json:"greets"`
}

type CreateGreetRequest struct {
	ProdId  uint
	GroupId uint
	Note    string
}

type CreateGreetResponse struct {
	ID uint
}

type DeleteGreetResponse struct {
	Rows int64
}

type StatsResponse struct {
	TotalGreets     int64 `json:"TotalGreets"`
	TotalProds      int64 `json:"TotalProds"`
	TotalGroups     int64 `json:"TotalGroups"`
	ProdsWithGreets int64 `json:"ProdsWithGreets"`
	GreetedGroups   int64 `json:"GreetedGroups"`
}

func makeRequest[T any](t *testing.T, url string, expectedStatus int) T {
	t.Helper()

	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, expectedStatus, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result T
	err = json.Unmarshal(bodyBytes, &result)
	require.NoError(t, err)

	return result
}

func makeRequestWithBody[T any](t *testing.T, url string, method string, body interface{}, expectedStatus int) T {
	t.Helper()

	jsonBody, err := json.Marshal(body)
	require.NoError(t, err)

	req, err := http.NewRequest(method, url, bytes.NewReader(jsonBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, expectedStatus, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result T
	err = json.Unmarshal(bodyBytes, &result)
	require.NoError(t, err)

	return result
}

func makeDeleteRequest[T any](t *testing.T, url string, expectedStatus int) T {
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	require.NoError(t, err)

	client := &http.Client{}
	resp, err := client.Do(req)
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

func TestPouetData(t *testing.T) {
	db, err := SetupDatabase(SetupArgs{
		DBFile:      ":memory:?cache=shared",
		Create:      true,
		PouetProds:  "./test/pouet-prods.json.gz",
		PouetGroups: "./test/pouet-groups.json.gz",
		BuildIndex:  true,
	})
	require.NoError(t, err)

	server := httptest.NewServer(Server(db, ""))
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

func TestGreets(t *testing.T) {
	db, err := SetupDatabase(SetupArgs{
		DBFile:      ":memory:?cache=shared",
		Create:      true,
		PouetProds:  "./test/pouet-prods.json.gz",
		PouetGroups: "./test/pouet-groups.json.gz",
		BuildIndex:  true,
	})
	require.NoError(t, err)

	server := httptest.NewServer(Server(db, ""))
	defer server.Close()

	expectedStats := StatsResponse{
		TotalGreets:     0,
		TotalProds:      420,
		TotalGroups:     64,
		ProdsWithGreets: 0,
		GreetedGroups:   0,
	}

	t.Run("PostNewGreet", func(t *testing.T) {
		stats := makeRequest[StatsResponse](t, server.URL+"/v1/stats", http.StatusOK)
		assert.Equal(t, expectedStats, stats)

		prodToGreet := uint(1)
		groupToGreet := uint(1)

		createReq := CreateGreetRequest{
			ProdId:  prodToGreet,
			GroupId: groupToGreet,
			Note:    "Test greet",
		}
		createResp := makeRequestWithBody[CreateGreetResponse](t, server.URL+"/v1/greets", http.MethodPost, createReq, http.StatusOK)
		assert.Greater(t, createResp.ID, uint(0))

		expectedStats.ProdsWithGreets += 1
		expectedStats.GreetedGroups += 1
		expectedStats.TotalGreets += 1

		stats = makeRequest[StatsResponse](t, server.URL+"/v1/stats", http.StatusOK)
		assert.Equal(t, expectedStats, stats)

		prodResp := makeRequest[ProdGetResponse](t, fmt.Sprintf("%s/v1/prods/%d", server.URL, prodToGreet), http.StatusOK)
		require.Len(t, prodResp.Greets, 1)
		assert.Equal(t, createResp.ID, prodResp.Greets[0].ID)
		assert.Equal(t, groupToGreet, prodResp.Greets[0].Group.ID)
		assert.Equal(t, "Test greet", prodResp.Greets[0].Note)

		groupGreets := makeRequest[[]GroupGreetsResponse](t, server.URL+"/v1/groups/1/greets", http.StatusOK)
		require.Len(t, groupGreets, 1)
		assert.Equal(t, prodToGreet, groupGreets[0].Prod.ID)
		assert.Equal(t, "Test greet", groupGreets[0].Reference)

		groupsGreeted := makeRequest[[]GroupGreetedResponse](t, server.URL+"/v1/groups/greeted", http.StatusOK)
		require.Len(t, groupsGreeted, 1)
		assert.Equal(t, groupToGreet, groupsGreeted[0].GroupID)
		assert.Equal(t, int64(1), groupsGreeted[0].Count)

		prodGreets := makeRequest[[]ProdGreetsResponse](t, server.URL+"/v1/prods/1/greets", http.StatusOK)
		require.Len(t, prodGreets, 1)
		assert.Equal(t, groupToGreet, prodGreets[0].GreeteeID)
		assert.Equal(t, "Test greet", prodGreets[0].Reference)
	})

	t.Run("PostNewGreetWithEmptyNote", func(t *testing.T) {
		createReq := CreateGreetRequest{
			ProdId:  2,
			GroupId: 1,
			Note:    "",
		}
		createResp := makeRequestWithBody[CreateGreetResponse](t, server.URL+"/v1/greets", http.MethodPost, createReq, http.StatusOK)
		assert.Greater(t, createResp.ID, uint(0))

		expectedStats.ProdsWithGreets += 1
		expectedStats.TotalGreets += 1

		stats := makeRequest[StatsResponse](t, server.URL+"/v1/stats", http.StatusOK)
		assert.Equal(t, expectedStats, stats)
	})

	/* FIXME these tests are broken for now, need database query fixes
	t.Run("POST non-existent prod", func(t *testing.T) {
		createReq := CreateGreetRequest{
			ProdId:  999999,
			GroupId: 1,
			Note:    "Should fail",
		}
		_ = makeRequestWithBody[struct{ Error string }](t, server.URL+"/v1/greets", http.MethodPost, createReq, http.StatusBadRequest)
	})

	t.Run("POST non-existent group", func(t *testing.T) {
		createReq := CreateGreetRequest{
			ProdId:  1,
			GroupId: 999999,
			Note:    "Should fail",
		}
		_ = makeRequestWithBody[struct{ Error string }](t, server.URL+"/v1/greets", http.MethodPost, createReq, http.StatusBadRequest)
	})
	*/

	t.Run("POST duplicate greet", func(t *testing.T) {
		createReq := CreateGreetRequest{
			ProdId:  1,
			GroupId: 1,
			Note:    "Duplicate",
		}
		_ = makeRequestWithBody[struct{ Error string }](t, server.URL+"/v1/greets", http.MethodPost, createReq, http.StatusBadRequest)
	})

	t.Run("POST malformed JSON", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/greets", bytes.NewReader([]byte("{invalid json")))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("POST empty request body", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/greets", bytes.NewReader([]byte{}))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("DELETE valid greet", func(t *testing.T) {
		// Use unique pair ofr prod id and group id to make sure that stats update is easy
		createReq := CreateGreetRequest{
			ProdId:  3,
			GroupId: 2,
			Note:    "Test greet",
		}
		createResp := makeRequestWithBody[CreateGreetResponse](t, server.URL+"/v1/greets", http.MethodPost, createReq, http.StatusOK)
		assert.Greater(t, createResp.ID, uint(0))

		expectedStats.ProdsWithGreets += 1
		expectedStats.GreetedGroups += 1
		expectedStats.TotalGreets += 1

		stats := makeRequest[StatsResponse](t, server.URL+"/v1/stats", http.StatusOK)
		assert.Equal(t, expectedStats, stats)

		deleteResp := makeDeleteRequest[DeleteGreetResponse](t, fmt.Sprintf("%s/v1/greets/%d", server.URL, createResp.ID), http.StatusOK)
		assert.Equal(t, int64(1), deleteResp.Rows)

		expectedStats.ProdsWithGreets -= 1
		expectedStats.GreetedGroups -= 1
		expectedStats.TotalGreets -= 1

		stats = makeRequest[StatsResponse](t, server.URL+"/v1/stats", http.StatusOK)
		assert.Equal(t, expectedStats, stats)
	})

	/* FIXME this test is broken, need db fixes
	t.Run("DELETE non-existent greet", func(t *testing.T) {
		deleteResp := makeDeleteRequest[DeleteGreetResponse](t, server.URL+"/v1/greets/999999", http.StatusNotFound)
		assert.Equal(t, DeleteGreetResponse{}, deleteResp)
	})
	*/

	t.Run("DELETE invalid ID format", func(t *testing.T) {
		deleteResp := makeDeleteRequest[DeleteGreetResponse](t, server.URL+"/v1/greets/abc", http.StatusBadRequest)
		assert.Equal(t, DeleteGreetResponse{}, deleteResp)
	})
}
