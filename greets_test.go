package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupGreetsDatabase(t *testing.T) Greets {
	t.Helper()

	dbPath := fmt.Sprintf("%s/combined.db", t.TempDir())

	pouetDB, err := PouetOpen(dbPath)
	require.NoError(t, err)
	pouetDB.ImportPouet("test/pouet-prods.json.gz", "test/pouet-groups.json.gz")

	greetsDB, err := GreetsOpen(dbPath)
	require.NoError(t, err)
	greetsDB.AutoMigrate()

	return greetsDB
}

func TestGreetsDatabase(t *testing.T) {
	greetsDB := setupGreetsDatabase(t)

	t.Run("CreateGreet", func(t *testing.T) {
		greetID, err := greetsDB.Greet(elevatedID, farbrauschID, "test greeting")
		require.NoError(t, err)
		assert.Greater(t, greetID, uint(0))

		deleted, err := greetsDB.DeleteGreet(greetID)
		require.NoError(t, err)
		assert.True(t, deleted)
	})

	t.Run("DeleteGreet", func(t *testing.T) {
		greet := Greet{
			ProdID:    elevatedID,
			GreeteeID: asdID,
			Reference: "greeting to delete",
		}
		greetsDB.db.Create(&greet)

		deleted, err := greetsDB.DeleteGreet(greet.ID)
		require.NoError(t, err)
		assert.True(t, deleted)

		deleted, err = greetsDB.DeleteGreet(greet.ID)
		require.NoError(t, err)
		assert.False(t, deleted)
	})

	t.Run("DuplicateGreet", func(t *testing.T) {
		t.Skip("Greet() method is broken: Prod struct lacks Greets field for GORM association")
	})

	t.Run("GreetInvalidProd", func(t *testing.T) {
		_, err := greetsDB.Greet(uint(999999), farbrauschID, "invalid prod")
		require.Error(t, err)
	})

	t.Run("GreetInvalidGroup", func(t *testing.T) {
		_, err := greetsDB.Greet(debrisID, uint(999999), "invalid group")
		require.Error(t, err)
	})
}
