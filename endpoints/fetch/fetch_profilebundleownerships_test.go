package fetch

import (
	"testing"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/stretchr/testify/require"
)

func Test_SummarizeBundleOwnerships(t *testing.T) {
	conn, err := sqlite.OpenConn("file::memory:?mode=memory", 0)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, models.HadesContext().AutoMigrate(conn))

	profileID := int64(1)
	ft := models.FetchTargetForProfileBundleOwnerships(profileID)

	// profile owns bundles 10 and 20 (20 twice: duplicate purchases dedupe)
	models.MustSave(conn, &itchio.BundleKey{ID: 1000, BundleID: 10, OwnerID: profileID})
	models.MustSave(conn, &itchio.BundleKey{ID: 1001, BundleID: 20, OwnerID: profileID})
	models.MustSave(conn, &itchio.BundleKey{ID: 1002, BundleID: 20, OwnerID: profileID})

	summarize := func() *butlerd.FetchProfileBundleOwnershipsResult {
		res := &butlerd.FetchProfileBundleOwnershipsResult{}
		summarizeBundleOwnerships(conn, profileID, ft, res)
		return res
	}

	// nothing synced yet
	res := summarize()
	require.EqualValues(t, 2, res.TotalBundles)
	require.EqualValues(t, 0, res.SyncedBundles)
	require.True(t, res.Stale)

	// everything fresh
	models.FetchTargetForProfileOwnedBundles(profileID).MustMarkFresh(conn)
	ft.MustMarkFresh(conn)
	models.FetchTargetForBundleGames(10).MustMarkFresh(conn)
	models.FetchTargetForBundleGames(20).MustMarkFresh(conn)
	res = summarize()
	require.EqualValues(t, 2, res.TotalBundles)
	require.EqualValues(t, 2, res.SyncedBundles)
	require.False(t, res.Stale)

	// one bundle's membership expired (e.g. a truncated walk):
	// partial count, stale overall
	models.FetchTargetForBundleGames(20).MustExpire(conn)
	res = summarize()
	require.EqualValues(t, 2, res.TotalBundles)
	require.EqualValues(t, 1, res.SyncedBundles)
	require.True(t, res.Stale)

	// profile-level target stale by itself also reports stale
	models.FetchTargetForBundleGames(20).MustMarkFresh(conn)
	ft.MustExpire(conn)
	res = summarize()
	require.EqualValues(t, 2, res.SyncedBundles)
	require.True(t, res.Stale)
}
