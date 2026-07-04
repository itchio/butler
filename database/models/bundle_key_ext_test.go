package models

import (
	"testing"

	"crawshaw.io/sqlite"
	itchio "github.com/itchio/go-itchio"
	"github.com/stretchr/testify/require"
)

func bundleTestConn(t *testing.T) *sqlite.Conn {
	conn, err := sqlite.OpenConn("file::memory:?mode=memory", 0)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, HadesContext().AutoMigrate(conn))
	return conn
}

// Seeds two profiles' bundle ownership:
//   - profile 1 owns bundle 10 twice (duplicate purchase, keys 1000 & 1001);
//     bundle 10 contains games 100 and 101
//   - profile 2 owns bundle 20 (key 2000); bundle 20 contains games 101 and 102
func seedBundleOwnership(t *testing.T, conn *sqlite.Conn) {
	MustSave(conn, &itchio.Bundle{ID: 10, Title: "Bundle Ten", GamesCount: 2})
	MustSave(conn, &itchio.Bundle{ID: 20, Title: "Bundle Twenty", GamesCount: 2})

	MustSave(conn, &itchio.BundleGame{BundleID: 10, GameID: 100, Position: 0})
	MustSave(conn, &itchio.BundleGame{BundleID: 10, GameID: 101, Position: 1})
	MustSave(conn, &itchio.BundleGame{BundleID: 20, GameID: 101, Position: 0})
	MustSave(conn, &itchio.BundleGame{BundleID: 20, GameID: 102, Position: 1})

	MustSave(conn, &itchio.BundleKey{ID: 1000, BundleID: 10, OwnerID: 1})
	MustSave(conn, &itchio.BundleKey{ID: 1001, BundleID: 10, OwnerID: 1})
	MustSave(conn, &itchio.BundleKey{ID: 2000, BundleID: 20, OwnerID: 2})
}

func Test_ProfileOwnsGameViaBundle(t *testing.T) {
	conn := bundleTestConn(t)
	seedBundleOwnership(t, conn)

	require.True(t, ProfileOwnsGameViaBundle(conn, 1, 100))
	require.True(t, ProfileOwnsGameViaBundle(conn, 1, 101))
	require.False(t, ProfileOwnsGameViaBundle(conn, 1, 102))

	require.False(t, ProfileOwnsGameViaBundle(conn, 2, 100))
	require.True(t, ProfileOwnsGameViaBundle(conn, 2, 102))

	require.False(t, ProfileOwnsGameViaBundle(conn, 3, 100))
	require.False(t, ProfileOwnsGameViaBundle(conn, 1, 999))
}

func Test_BundleIDOwningGameForProfile(t *testing.T) {
	conn := bundleTestConn(t)
	seedBundleOwnership(t, conn)

	require.EqualValues(t, 10, BundleIDOwningGameForProfile(conn, 100, 1))
	require.EqualValues(t, 0, BundleIDOwningGameForProfile(conn, 100, 2))
	require.EqualValues(t, 20, BundleIDOwningGameForProfile(conn, 102, 2))
	require.EqualValues(t, 0, BundleIDOwningGameForProfile(conn, 999, 1))
}

func Test_BundleIDOwningGameAnyProfile(t *testing.T) {
	conn := bundleTestConn(t)
	seedBundleOwnership(t, conn)

	// game 100 is only in bundle 10, owned by profile 1
	bundleID, profileID := BundleIDOwningGameAnyProfile(conn, 100)
	require.EqualValues(t, 10, bundleID)
	require.EqualValues(t, 1, profileID)

	// game 101 is in both bundles; the returned pair must be consistent
	bundleID, profileID = BundleIDOwningGameAnyProfile(conn, 101)
	switch bundleID {
	case 10:
		require.EqualValues(t, 1, profileID)
	case 20:
		require.EqualValues(t, 2, profileID)
	default:
		t.Fatalf("unexpected bundleID %d", bundleID)
	}

	bundleID, profileID = BundleIDOwningGameAnyProfile(conn, 999)
	require.EqualValues(t, 0, bundleID)
	require.EqualValues(t, 0, profileID)
}

func Test_BundleKeysByGameID(t *testing.T) {
	conn := bundleTestConn(t)
	seedBundleOwnership(t, conn)

	// game 101 is in bundle 10 (two keys, duplicate purchase) and
	// bundle 20 (one key)
	keys := BundleKeysByGameID(conn, 101)
	var ids []int64
	for _, bk := range keys {
		ids = append(ids, bk.ID)
	}
	require.ElementsMatch(t, []int64{1000, 1001, 2000}, ids)

	require.Empty(t, BundleKeysByGameID(conn, 999))
}

func Test_DistinctOwnedBundleIDs(t *testing.T) {
	conn := bundleTestConn(t)
	seedBundleOwnership(t, conn)

	// duplicate purchases of bundle 10 dedupe to a single ID
	require.ElementsMatch(t, []int64{10}, DistinctOwnedBundleIDs(conn, 1))
	require.ElementsMatch(t, []int64{20}, DistinctOwnedBundleIDs(conn, 2))
	require.Empty(t, DistinctOwnedBundleIDs(conn, 3))
}
