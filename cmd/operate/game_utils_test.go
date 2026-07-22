package operate

import (
	"testing"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func accessTestConn(t *testing.T) *sqlite.Conn {
	conn, err := sqlite.OpenConn("file::memory:?mode=memory", 0)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, models.HadesContext().AutoMigrate(conn))
	return conn
}

func TestStrictAccessMissingProfile(t *testing.T) {
	conn := accessTestConn(t)
	_, err := StrictAccessForGameID(conn, 100, 999)
	require.Equal(t, butlerd.CodeNoSuchProfile, errors.Cause(err))
}

func TestStrictAccessUnownedGame(t *testing.T) {
	conn := accessTestConn(t)
	models.MustSave(conn, &models.Profile{ID: 1, APIKey: "key1"})

	access, err := StrictAccessForGameID(conn, 100, 1)
	require.NoError(t, err)
	require.Equal(t, "key1", access.APIKey)
	require.EqualValues(t, 1, access.ProfileID)
	require.Zero(t, access.Credentials.DownloadKeyID)
}

func TestStrictAccessWithDownloadKey(t *testing.T) {
	conn := accessTestConn(t)
	models.MustSave(conn, &models.Profile{ID: 1, APIKey: "key1"})
	models.MustSave(conn, &itchio.DownloadKey{ID: 500, GameID: 100, OwnerID: 1})

	access, err := StrictAccessForGameID(conn, 100, 1)
	require.NoError(t, err)
	require.EqualValues(t, 500, access.Credentials.DownloadKeyID)
}

func TestStrictAccessNeverFallsBack(t *testing.T) {
	conn := accessTestConn(t)
	models.MustSave(conn, &models.Profile{ID: 1, APIKey: "key1"})
	models.MustSave(conn, &models.Profile{ID: 2, APIKey: "key2"})
	models.MustSave(conn, &itchio.DownloadKey{ID: 500, GameID: 100, OwnerID: 2})

	access, err := StrictAccessForGameID(conn, 100, 1)
	require.NoError(t, err)
	require.Equal(t, "key1", access.APIKey)
	require.Zero(t, access.Credentials.DownloadKeyID)
}
