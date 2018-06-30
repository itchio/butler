package hades

import (
	"fmt"
	"reflect"

	"crawshaw.io/sqlite"
	"github.com/pkg/errors"
)

func (c *Context) saveJoins(conn *sqlite.Conn, mode AssocMode, mtm *ManyToMany) error {
	selectQuery := fmt.Sprintf(`SELECT %s FROM %s WHERE %s = ?`,
		EscapeIdentifier(mtm.DestinDBName),
		EscapeIdentifier(mtm.JoinTable),
		EscapeIdentifier(mtm.SourceDBName),
	)
	upsertQuery := fmt.Sprintf(`INSERT INTO %s (%s, %s) VALUES (?, ?) ON CONFLICT DO NOTHING`,
		EscapeIdentifier(mtm.JoinTable),
		EscapeIdentifier(mtm.SourceDBName),
		EscapeIdentifier(mtm.DestinDBName),
	)
	deleteQuery := fmt.Sprintf(`DELETE FROM %s WHERE %s = ? AND %s = ?`,
		EscapeIdentifier(mtm.JoinTable),
		EscapeIdentifier(mtm.SourceDBName),
		EscapeIdentifier(mtm.DestinDBName),
	)
	deleteAllQuery := fmt.Sprintf(`DELETE FROM %s WHERE %s = ?`,
		EscapeIdentifier(mtm.JoinTable),
		EscapeIdentifier(mtm.SourceDBName),
	)

	for sourceKey, joinRecs := range mtm.Values {
		for _, jr := range joinRecs {
			if jr.Record.IsValid() {
				// many to many record was specified
				err := c.Upsert(conn, mtm.Scope, jr.Record)
				if err != nil {
					return err
				}
			} else {
				// create our own many to many record
				err := c.ExecRaw(conn, upsertQuery, nil,
					sourceKey, jr.DestinKey,
				)
				if err != nil {
					return err
				}
			}
		}

		if mode == AssocModeReplace {
			// this essentially clears all associated records
			if len(joinRecs) == 0 {
				err := c.ExecRaw(conn, deleteAllQuery, nil, sourceKey)
				if err != nil {
					return err
				}
				continue
			}

			passedDKs := make(map[interface{}]struct{})
			for _, jr := range joinRecs {
				passedDKs[jr.DestinKey] = struct{}{}
			}

			// we have > 0 joinRecs, as checked above
			firstDK := joinRecs[0].DestinKey
			dkTyp := reflect.TypeOf(firstDK)
			dkKind := dkTyp.Kind()

			var removedDKs []interface{}
			{
				err := c.ExecRaw(conn, selectQuery, func(stmt *sqlite.Stmt) error {
					var dk interface{}
					switch dkKind {
					case reflect.Int64:
						dk = stmt.ColumnInt64(0)
					case reflect.String:
						dk = stmt.ColumnText(0)
					default:
						return errors.Errorf("Unsupported primary key for join table: %v", dkTyp)
					}

					if _, ok := passedDKs[dk]; !ok {
						removedDKs = append(removedDKs, dk)
					}
					return nil
				}, sourceKey)
				if err != nil {
					return err
				}
			}

			for _, dk := range removedDKs {
				err := c.ExecRaw(conn, deleteQuery, nil, sourceKey, dk)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
