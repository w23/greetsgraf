package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
)

func setupPouetDatabase(t *testing.T, name string) Pouet {
	t.Helper()

	pouetDB, err := PouetOpen(fmt.Sprintf("%s/%s.db", t.TempDir(), name))
	require.NoError(t, err)

	// Load initial test data used for all tests
	// Use pouet.sh script to update/populate these files
	pouetDB.ImportPouet("test/pouet-prods.json.gz", "test/pouet-groups.json.gz")

	return pouetDB
}

const debrisID = uint(30244)
const farbrauschID = uint(322)
const asdID = uint(1317)
const elevatedID = uint(52938)
const rgbaID = uint(697)
const tbcID = uint(1623)

func TestPouetImport(t *testing.T) {
	db := setupPouetDatabase(t, "pouet-import")

	// Check that basic loading went fine
	t.Run("DebrisWasImported", func(t *testing.T) {
		debris, err := db.GetProd(debrisID)
		require.NoError(t, err)
		require.Len(t, debris.Groups, 1)
		assert.Equal(t, farbrauschID, debris.Groups[0].ID)
		assert.Equal(t, "fr-041: debris.", debris.Name)
		assert.Equal(t, 4, debris.Month)
		assert.Equal(t, 2007, debris.Year)
	})

	t.Run("ASDWasImported", func(t *testing.T) {
		asd, err := db.GetGroup(asdID)
		require.NoError(t, err)
		assert.Equal(t, "Andromeda Software Development", asd.Name)
	})

	// Prods with multiple groups should be reported as such
	t.Run("Multigroup", func(t *testing.T) {
		elevated, err := db.GetProd(elevatedID)
		require.NoError(t, err)
		require.Equal(t, "elevated", elevated.Name)

		groupIDs := []uint{}
		for _, group := range elevated.Groups {
			groupIDs = append(groupIDs, group.ID)
		}
		assert.ElementsMatch(t, groupIDs, []uint{rgbaID, tbcID})
	})

	t.Run("ParseYearOnlyDates", func(t *testing.T) {
		// `asdtro` has weird date value, make sure it is properly parsed
		prod, err := db.GetProd(4961)
		require.NoError(t, err)
		assert.Equal(t, 1993, prod.Year)
		assert.Equal(t, 0, prod.Month)
	})

	// TODO:
	// - fuzzy prod search
	// - fuzzy group search
}

// TODO greetings test:
// - add
// - list greets for prod
// - list greets of a group
// - remove
// - rank
// - stats
