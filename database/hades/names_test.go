package hades_test

import (
	"testing"

	"github.com/jinzhu/gorm"

	"github.com/alecthomas/assert"
	"github.com/itchio/butler/database/hades"
)

func Test_FromDBName(t *testing.T) {
	assert.EqualValues(t, "OwnerID", hades.FromDBName("owner_id"))
	assert.EqualValues(t, "ID", hades.FromDBName("id"))
	assert.EqualValues(t, "ProfileGames", hades.FromDBName("profile_games"))
}

func Test_ToDBName(t *testing.T) {
	assert.EqualValues(t, "owner_id", gorm.ToDBName("OwnerId"))
	assert.EqualValues(t, "id", gorm.ToDBName("ID"))
	assert.EqualValues(t, "profile_games", gorm.ToDBName("ProfileGames"))
}
