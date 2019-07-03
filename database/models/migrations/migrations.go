package migrations

import (
	"sort"
	"time"

	"xorm.io/builder"
	"github.com/itchio/butler/butlerd/horror"
	"github.com/itchio/hades"
	"github.com/pkg/errors"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqliteutil"
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
			defer sqliteutil.Save(conn)(&retErr)
			err := migration(consumer, conn)
			if err != nil {
				return err
			}
			models.SetSchemaVersion(conn, key)
			return nil
		}()
		return errors.Wrapf(err, "While running migration %d", key)
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
