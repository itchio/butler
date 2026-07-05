package fetch

import (
	"testing"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/stretchr/testify/require"
)

// The renderer programs its CTA state against this matrix, so pin it:
// (ownership source × bundle sync freshness) → (Owned, Source, Stale).
func Test_GameOwnershipMatrix(t *testing.T) {
	conn, err := sqlite.OpenConn("file::memory:?mode=memory", 0)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, models.HadesContext().AutoMigrate(conn))

	profileID := int64(1)

	// game 100: owned via download key; game 200: owned via bundle 10;
	// game 300: not owned
	models.MustSave(conn, &itchio.DownloadKey{ID: 5000, GameID: 100, OwnerID: profileID})
	models.MustSave(conn, &itchio.BundleKey{ID: 1000, BundleID: 10, OwnerID: profileID})
	models.MustSave(conn, &itchio.BundleGame{BundleID: 10, GameID: 200})

	ownedBundles := models.FetchTargetForProfileOwnedBundles(profileID)
	ownerships := models.FetchTargetForProfileBundleOwnerships(profileID)

	check := func(gameID int64, fresh bool) *butlerd.FetchGameOwnershipResult {
		if fresh {
			ownedBundles.MustMarkFresh(conn)
			ownerships.MustMarkFresh(conn)
		} else {
			ownedBundles.MustExpire(conn)
			ownerships.MustExpire(conn)
		}
		res := &butlerd.FetchGameOwnershipResult{}
		gameOwnershipFromConn(conn, profileID, gameID, res)
		return res
	}

	// download key: owned, never stale, regardless of bundle sync state
	for _, fresh := range []bool{true, false} {
		res := check(100, fresh)
		require.True(t, res.Owned)
		require.EqualValues(t, "download_key", res.Source)
		require.EqualValues(t, 5000, res.DownloadKeyID)
		require.False(t, res.Stale)
	}

	// bundle-owned: owned either way, stale tracks the sync targets
	res := check(200, true)
	require.True(t, res.Owned)
	require.EqualValues(t, "bundle", res.Source)
	require.EqualValues(t, 10, res.BundleID)
	require.False(t, res.Stale)

	res = check(200, false)
	require.True(t, res.Owned, "stale cached positive still reported")
	require.EqualValues(t, "bundle", res.Source)
	require.True(t, res.Stale)

	// not owned: fresh index means a definitive no, stale index means
	// "not locally known yet"
	res = check(300, true)
	require.False(t, res.Owned)
	require.Empty(t, res.Source)
	require.False(t, res.Stale)

	res = check(300, false)
	require.False(t, res.Owned)
	require.True(t, res.Stale)

	// another profile owns nothing here even with fresh targets
	res = &butlerd.FetchGameOwnershipResult{}
	gameOwnershipFromConn(conn, 2, 200, res)
	require.False(t, res.Owned)
}
