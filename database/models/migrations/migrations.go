package migrations

import (
	"sort"
	"time"

	"github.com/itchio/butler/butlerd/horror"
	"github.com/itchio/hades"
	"github.com/pkg/errors"
	"xorm.io/builder"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/headway/state"
)

type Migration func(consumer *state.Consumer, conn *sqlite.Conn) error

var migrations = map[int64]Migration{
	// create "cave_historical_playtime" records from all caves so far
	1542741863: func(consumer *state.Consumer, conn *sqlite.Conn) error {
		var caves []*models.Cave
		models.MustSelect(conn, &caves, builder.NewCond(), hades.Search{})

		var playtimes []*models.CaveHistoricalPlayTime
		for _, cave := range caves {
			if cave.SecondsRun > 0 {
				now := time.Now().UTC()
				lastTouchedAt := cave.LastTouchedAt
				if lastTouchedAt == nil {
					lastTouchedAt = cave.InstalledAt
				}
				if lastTouchedAt == nil {
					lastTouchedAt = &now
				}

				playtimes = append(playtimes, &models.CaveHistoricalPlayTime{
					CaveID:        cave.ID,
					GameID:        cave.GameID,
					UploadID:      cave.UploadID,
					BuildID:       cave.BuildID,
					SecondsRun:    cave.SecondsRun,
					LastTouchedAt: lastTouchedAt,
					CreatedAt:     &now,
				})
			}
		}
		consumer.Infof("Saving %d historical playtimes", len(playtimes))
		models.MustSave(conn, playtimes)

		return nil
	},
	// add explicit lookup indexes for bundle ownership tables
	1747200000: func(consumer *state.Consumer, conn *sqlite.Conn) error {
		stmts := []string{
			"CREATE INDEX IF NOT EXISTS idx_bundle_keys_owner_id ON bundle_keys(owner_id)",
			"CREATE INDEX IF NOT EXISTS idx_bundle_keys_owner_bundle ON bundle_keys(owner_id, bundle_id)",
			"CREATE INDEX IF NOT EXISTS idx_bundle_games_game_bundle ON bundle_games(game_id, bundle_id)",
			"CREATE INDEX IF NOT EXISTS idx_bundle_games_bundle_game ON bundle_games(bundle_id, game_id)",
		}
		for _, s := range stmts {
			err := sqlitex.Exec(conn, s, nil)
			if err != nil {
				return errors.Wrapf(err, "executing %q", s)
			}
		}
		return nil
	},
}

func Do(consumer *state.Consumer, conn *sqlite.Conn) error {
	currentVersion := models.GetSchemaVersion(conn)
	consumer.Debugf("Current DB version is %d", currentVersion)
	consumer.Debugf("Latest migration is   %d", LatestSchemaVersion())

	todo := getKeysAfter(currentVersion)
	if len(todo) == 0 {
		consumer.Debugf("No migrations to run")
		return nil
	}

	consumer.Debugf("%d migrations to run (%v)", len(todo), todo)
	for _, key := range todo {
		consumer.Debugf("Running migration %d...", key)
		migration := migrations[key]
		err := func() (retErr error) {
			defer horror.RecoverInto(&retErr)
			// run migration in a transaction
			defer sqlitex.Save(conn)(&retErr)
			err := migration(consumer, conn)
			if err != nil {
				return err
			}
			models.SetSchemaVersion(conn, key)
			return nil
		}()
		if err != nil {
			return errors.Wrapf(err, "While running migration %d", key)
		}
	}

	return nil
}

var sortedKeys []int64

func getSortedKeys() []int64 {
	if sortedKeys == nil {
		var keys []int64
		for k := range migrations {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i int, j int) bool {
			return keys[i] < keys[j]
		})
		sortedKeys = keys
	}
	return sortedKeys
}

func getKeysAfter(version int64) []int64 {
	var result []int64
	for _, k := range getSortedKeys() {
		if k > version {
			result = append(result, k)
		}
	}
	return result
}

func LatestSchemaVersion() int64 {
	keys := getSortedKeys()
	if len(keys) == 0 {
		return 0
	}
	return keys[len(keys)-1]
}
