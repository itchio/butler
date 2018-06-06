package hades

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"crawshaw.io/sqlite"
	"github.com/itchio/hades/sqliteutil2"
	"github.com/pkg/errors"
)

type AutoMigrateStats struct {
	NumCreated  int64
	NumMigrated int64
	NumCurrent  int64
}

func (c *Context) AutoMigrate(conn *sqlite.Conn) error {
	return c.AutoMigrateEx(conn, &AutoMigrateStats{})
}

func (c *Context) AutoMigrateEx(conn *sqlite.Conn, stats *AutoMigrateStats) error {
	for _, m := range c.ScopeMap.byDBName {
		err := c.syncTable(conn, stats, m.GetModelStruct())
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Context) syncTable(conn *sqlite.Conn, stats *AutoMigrateStats, ms *ModelStruct) (err error) {
	tableName := ms.TableName
	pti, err := c.PragmaTableInfo(conn, tableName)
	if err != nil {
		return err
	}
	if len(pti) == 0 {
		stats.NumCreated++
		return c.createTable(conn, ms)
	}

	// migrate table in transaction
	defer sqliteutil2.Save(conn)(&err)

	err = c.ExecRaw(conn, "PRAGMA foreign_keys = 0", nil)
	if err != nil {
		return nil
	}

	oldColumns := make(map[string]PragmaTableInfoRow)
	for _, ptir := range pti {
		oldColumns[ptir.Name] = ptir
	}

	numOldCols := len(oldColumns)
	numNewCols := 0
	isMissingCols := false

	{
		var processField func(sf *StructField)
		processField = func(sf *StructField) {
			if sf.IsSquashed {
				for _, nsf := range sf.SquashedFields {
					processField(nsf)
				}
			}

			if !sf.IsNormal {
				return
			}
			numNewCols++

			if _, ok := oldColumns[sf.DBName]; !ok {
				isMissingCols = true
				return
			}
		}
		for _, sf := range ms.StructFields {
			processField(sf)
		}
	}

	if !isMissingCols && numOldCols == numNewCols {
		// all done
		stats.NumCurrent++
		return nil
	}

	stats.NumMigrated++
	tempName := fmt.Sprintf("__hades_migrate__%s__", tableName)
	err = c.ExecRaw(conn, fmt.Sprintf("CREATE TABLE %s AS SELECT * FROM %s", tempName, tableName), nil)
	if err != nil {
		return nil
	}

	err = c.dropTable(conn, tableName)
	if err != nil {
		return nil
	}

	err = c.createTable(conn, ms)
	if err != nil {
		return err
	}

	var columns []string
	{
		var processField func(sf *StructField)
		processField = func(sf *StructField) {
			if sf.IsSquashed {
				for _, nsf := range sf.SquashedFields {
					processField(nsf)
				}
			}

			if !sf.IsNormal {
				return
			}

			if _, ok := oldColumns[sf.DBName]; ok {
				columns = append(columns, EscapeIdentifier(sf.DBName))
			}
		}
		for _, sf := range ms.StructFields {
			processField(sf)
		}
	}
	var columnList = strings.Join(columns, ",")

	query := fmt.Sprintf("INSERT INTO %s (%s) SELECT %s FROM %s",
		tableName,
		columnList,
		columnList,
		tempName,
	)

	err = c.ExecRaw(conn, query, nil)
	if err != nil {
		return nil
	}

	err = c.dropTable(conn, tempName)
	if err != nil {
		return nil
	}

	err = c.ExecRaw(conn, "PRAGMA foreign_keys = 1", nil)
	if err != nil {
		return nil
	}

	return nil
}

func (c *Context) createTable(conn *sqlite.Conn, ms *ModelStruct) error {
	query := fmt.Sprintf("CREATE TABLE %s", EscapeIdentifier(ms.TableName))
	var columns []string
	var pks []string

	var processField func(sf *StructField) error
	processField = func(sf *StructField) error {
		if sf.IsSquashed {
			for _, nsf := range sf.SquashedFields {
				err := processField(nsf)
				if err != nil {
					return err
				}
			}
		}

		if !sf.IsNormal {
			return nil
		}

		var sqliteType string
		typ := sf.Struct.Type
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}

		switch typ.Kind() {
		case reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8, reflect.Int,
			reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8, reflect.Uint:
			sqliteType = "INTEGER"
		case reflect.Bool:
			sqliteType = "BOOLEAN"
		case reflect.Float64, reflect.Float32:
			sqliteType = "REAL"
		case reflect.String:
			sqliteType = "TEXT"
		case reflect.Struct:
			if typ == reflect.TypeOf(time.Time{}) {
				sqliteType = "DATETIME"
				break
			}
			fallthrough
		default:
			return errors.Errorf("Unsupported model field type: %v (in model %v)", sf.Struct.Type, ms.ModelType)
		}
		modifier := ""
		if sf.IsPrimaryKey {
			pks = append(pks, sf.DBName)
			modifier = " NOT NULL"
		}
		column := fmt.Sprintf(`%s %s%s`, EscapeIdentifier(sf.DBName), sqliteType, modifier)
		columns = append(columns, column)
		return nil
	}

	for _, sf := range ms.StructFields {
		err := processField(sf)
		if err != nil {
			return err
		}
	}

	if len(pks) > 0 {
		columns = append(columns, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(pks, ", ")))
	} else {
		return errors.Errorf("Model %v has no primary keys", ms.ModelType)
	}
	query = fmt.Sprintf("%s (%s)", query, strings.Join(columns, ", "))

	return c.ExecRaw(conn, query, nil)
}

func (c *Context) dropTable(conn *sqlite.Conn, tableName string) error {
	return c.ExecRaw(conn, fmt.Sprintf("DROP TABLE %s", EscapeIdentifier(tableName)), nil)
}
