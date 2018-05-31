package hades

import "reflect"

// JoinTableHandlerInterface is an interface for how to handle many_to_many relations
type JoinTableHandlerInterface interface {
	// initialize join table handler
	Setup(relationship *Relationship, tableName string, source reflect.Type, destination reflect.Type)
	// Table return join table's table name
	Table() string
	// SourceForeignKeys return source foreign keys
	SourceForeignKeys() []JoinTableForeignKey
	// DestinationForeignKeys return destination foreign keys
	DestinationForeignKeys() []JoinTableForeignKey
}

// JoinTableForeignKey join table foreign key struct
type JoinTableForeignKey struct {
	DBName            string
	AssociationDBName string
}

// JoinTableSource is a struct that contains model type and foreign keys
type JoinTableSource struct {
	ModelType   reflect.Type
	ForeignKeys []JoinTableForeignKey
}
