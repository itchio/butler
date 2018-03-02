package fetch

import (
	"fmt"
	"log"
	"reflect"

	"github.com/go-errors/errors"
	"github.com/jinzhu/gorm"
)

func diff(tx *gorm.DB, fresh []interface{}) error {
	first := fresh[0]
	log.Printf("typeof first indirect: %v", reflect.TypeOf(first).Elem())

	var primaryKey string = ""
	scope := tx.NewScope(first)
	fs := scope.Fields()
	for _, f := range fs {
		if f.IsPrimaryKey {
			log.Printf("found primary key: %s", f.Name)
			primaryKey = f.Name
		}
	}

	var keys []interface{}
	for _, record := range fresh {
		f := reflect.Indirect(reflect.ValueOf(record)).FieldByName(primaryKey).Interface()
		log.Printf("value of key = %#v", f)
		keys = append(keys, f)
	}

	var stale []interface{}
	err := tx.Table(scope.TableName()).Where(fmt.Sprintf("%s in (?)", primaryKey), keys).Find(&stale).Error
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
