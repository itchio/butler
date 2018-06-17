package hades

import (
	"fmt"
	"go/ast"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/jinzhu/inflection"
	"github.com/pkg/errors"
)

type safeModelStructsMap struct {
	m map[reflect.Type]*ModelStruct
	l *sync.RWMutex
}

func (s *safeModelStructsMap) Set(key reflect.Type, value *ModelStruct) {
	s.l.Lock()
	defer s.l.Unlock()
	s.m[key] = value
}

func (s *safeModelStructsMap) Get(key reflect.Type) *ModelStruct {
	s.l.RLock()
	defer s.l.RUnlock()
	return s.m[key]
}

func newModelStructsMap() *safeModelStructsMap {
	return &safeModelStructsMap{l: new(sync.RWMutex), m: make(map[reflect.Type]*ModelStruct)}
}

var modelStructsMap = newModelStructsMap()

// ModelStruct model definition
type ModelStruct struct {
	PrimaryFields      []*StructField
	StructFields       []*StructField
	StructFieldsByName map[string]*StructField
	ModelType          reflect.Type
	TableName          string
}

// StructField model field's struct definition
type StructField struct {
	DBName         string
	Name           string
	Names          []string
	IsPrimaryKey   bool
	IsNormal       bool
	IsIgnored      bool
	IsScanner      bool
	IsSquashed     bool
	SquashedFields []*StructField
	Tag            reflect.StructTag
	TagSettings    map[TagSetting]string
	Struct         reflect.StructField
	IsForeignKey   bool
	Relationship   *Relationship
}

// Relationship described the relationship between models
type Relationship struct {
	// belongs_to, has_one, has_many, many_to_many
	Kind                         string
	PolymorphicType              string
	PolymorphicDBName            string
	PolymorphicValue             string
	ForeignFieldNames            []string
	ForeignDBNames               []string
	AssociationForeignFieldNames []string
	AssociationForeignDBNames    []string
	JoinTableHandler             JoinTableHandlerInterface
}

func getForeignField(column string, fields []*StructField) *StructField {
	for _, field := range fields {
		if field.Name == column || field.DBName == column || field.DBName == ToDBName(column) {
			return field
		}
	}
	return nil
}

type TagSetting string

const (
	TagSettingIgnore                         TagSetting = "-"
	TagSettingSquash                         TagSetting = "squash"
	TagSettingManyToMany                     TagSetting = "many_to_many"
	TagSettingPrimaryKey                     TagSetting = "primary_key"
	TagSettingForeignKey                     TagSetting = "foreign_key"
	TagSettingAssociationForeignKey          TagSetting = "association_foreign_key"
	TagSettingJoinTableForeignKey            TagSetting = "join_table_foreign_key"
	TagSettingAssociationJoinTableForeignKey TagSetting = "association_join_table_foreign_key"
)

var ValidTagSettings = map[TagSetting]bool{
	TagSettingIgnore:                         true,
	TagSettingSquash:                         true,
	TagSettingManyToMany:                     true,
	TagSettingPrimaryKey:                     true,
	TagSettingForeignKey:                     true,
	TagSettingAssociationForeignKey:          true,
	TagSettingJoinTableForeignKey:            true,
	TagSettingAssociationJoinTableForeignKey: true,
}

// GetModelStruct get value's model struct, relationships based on struct and tag definition
func (scope *Scope) GetModelStruct() *ModelStruct {
	var modelStruct ModelStruct
	// Scope value can't be nil
	if scope.Value == nil {
		return &modelStruct
	}

	reflectType := reflect.ValueOf(scope.Value).Type()
	for reflectType.Kind() == reflect.Slice || reflectType.Kind() == reflect.Ptr {
		reflectType = reflectType.Elem()
	}

	// Scope value need to be a struct
	if reflectType.Kind() != reflect.Struct {
		return &modelStruct
	}

	// Get Cached model struct
	if value := modelStructsMap.Get(reflectType); value != nil {
		return value
	}

	modelStruct.TableName = TableName(reflectType)
	modelStruct.ModelType = reflectType
	modelStruct.StructFieldsByName = make(map[string]*StructField)

	// Get all fields
	for i := 0; i < reflectType.NumField(); i++ {
		if fieldStruct := reflectType.Field(i); ast.IsExported(fieldStruct.Name) {
			field := &StructField{
				Struct:      fieldStruct,
				Name:        fieldStruct.Name,
				Names:       []string{fieldStruct.Name},
				Tag:         fieldStruct.Tag,
				TagSettings: parseTagSetting(reflectType, fieldStruct.Name, fieldStruct.Tag),
			}

			// is ignored field
			if _, ok := field.TagSettings[TagSettingIgnore]; ok {
				field.IsIgnored = true
			} else if _, ok := field.TagSettings[TagSettingSquash]; ok {
				field.IsSquashed = true
				nestedModelStruct := scope.ctx.NewScope(reflect.Zero(field.Struct.Type).Interface()).GetModelStruct()
				field.SquashedFields = append(field.SquashedFields, nestedModelStruct.StructFields...)
			} else {
				if _, ok := field.TagSettings[TagSettingPrimaryKey]; ok {
					field.IsPrimaryKey = true
					modelStruct.PrimaryFields = append(modelStruct.PrimaryFields, field)
				}

				indirectType := fieldStruct.Type
				for indirectType.Kind() == reflect.Ptr {
					indirectType = indirectType.Elem()
				}

				fieldValue := reflect.New(indirectType).Interface()
				if _, isTime := fieldValue.(*time.Time); isTime {
					// is time
					field.IsNormal = true
				} else {
					// build relationships
					switch indirectType.Kind() {
					case reflect.Slice:
						defer func(field *StructField) {
							var (
								relationship           = &Relationship{}
								toScope                = scope.New(reflect.New(field.Struct.Type).Interface())
								foreignKeys            []string
								associationForeignKeys []string
								elemType               = field.Struct.Type
							)

							if foreignKey := field.TagSettings[TagSettingForeignKey]; foreignKey != "" {
								foreignKeys = strings.Split(foreignKey, ",")
							}

							if foreignKey := field.TagSettings[TagSettingAssociationForeignKey]; foreignKey != "" {
								associationForeignKeys = strings.Split(foreignKey, ",")
							}

							for elemType.Kind() == reflect.Slice || elemType.Kind() == reflect.Ptr {
								elemType = elemType.Elem()
							}

							if elemType.Kind() == reflect.Struct {
								if manyToMany := field.TagSettings[TagSettingManyToMany]; manyToMany != "" {
									relationship.Kind = "many_to_many"

									{ // Foreign Keys for Source
										joinTableDBNames := []string{}

										if foreignKey := field.TagSettings[TagSettingAssociationJoinTableForeignKey]; foreignKey != "" {
											joinTableDBNames = strings.Split(foreignKey, ",")
										}

										// if no foreign keys defined with tag
										if len(foreignKeys) == 0 {
											for _, field := range modelStruct.PrimaryFields {
												foreignKeys = append(foreignKeys, field.DBName)
											}
										}

										for idx, foreignKey := range foreignKeys {
											if foreignField := getForeignField(foreignKey, modelStruct.StructFields); foreignField != nil {
												// source foreign keys (db names)
												relationship.ForeignFieldNames = append(relationship.ForeignFieldNames, foreignField.DBName)

												// setup join table foreign keys for source
												if len(joinTableDBNames) > idx {
													// if defined join table's foreign key
													relationship.ForeignDBNames = append(relationship.ForeignDBNames, joinTableDBNames[idx])
												} else {
													defaultJointableForeignKey := ToDBName(reflectType.Name()) + "_" + foreignField.DBName
													relationship.ForeignDBNames = append(relationship.ForeignDBNames, defaultJointableForeignKey)
												}
											}
										}
									}

									{ // Foreign Keys for Association (Destination)
										associationJoinTableDBNames := []string{}

										if foreignKey := field.TagSettings[TagSettingAssociationJoinTableForeignKey]; foreignKey != "" {
											associationJoinTableDBNames = strings.Split(foreignKey, ",")
										}

										// if no association foreign keys defined with tag
										if len(associationForeignKeys) == 0 {
											for _, field := range toScope.PrimaryFields() {
												associationForeignKeys = append(associationForeignKeys, field.DBName)
											}
										}

										for idx, name := range associationForeignKeys {
											if field, ok := toScope.FieldByName(name); ok {
												// association foreign keys (db names)
												relationship.AssociationForeignFieldNames = append(relationship.AssociationForeignFieldNames, field.DBName)

												// setup join table foreign keys for association
												if len(associationJoinTableDBNames) > idx {
													relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, associationJoinTableDBNames[idx])
												} else {
													// join table foreign keys for association
													joinTableDBName := ToDBName(elemType.Name()) + "_" + field.DBName
													relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, joinTableDBName)
												}
											}
										}
									}

									joinTableHandler := JoinTableHandler{}
									joinTableHandler.Setup(relationship, manyToMany, reflectType, elemType)
									relationship.JoinTableHandler = &joinTableHandler
									field.Relationship = relationship
								} else {
									// User has many comments, associationType is User, comment use UserID as foreign key
									var associationType = reflectType.Name()
									var toFields = toScope.GetStructFields()
									relationship.Kind = "has_many"

									// if no foreign keys defined with tag
									if len(foreignKeys) == 0 {
										// if no association foreign keys defined with tag
										if len(associationForeignKeys) == 0 {
											for _, field := range modelStruct.PrimaryFields {
												foreignKeys = append(foreignKeys, associationType+field.Name)
												associationForeignKeys = append(associationForeignKeys, field.Name)
											}
										} else {
											// generate foreign keys from defined association foreign keys
											for _, scopeFieldName := range associationForeignKeys {
												if foreignField := getForeignField(scopeFieldName, modelStruct.StructFields); foreignField != nil {
													foreignKeys = append(foreignKeys, associationType+foreignField.Name)
													associationForeignKeys = append(associationForeignKeys, foreignField.Name)
												}
											}
										}
									} else {
										// generate association foreign keys from foreign keys
										if len(associationForeignKeys) == 0 {
											for _, foreignKey := range foreignKeys {
												if strings.HasPrefix(foreignKey, associationType) {
													associationForeignKey := strings.TrimPrefix(foreignKey, associationType)
													if foreignField := getForeignField(associationForeignKey, modelStruct.StructFields); foreignField != nil {
														associationForeignKeys = append(associationForeignKeys, associationForeignKey)
													}
												}
											}
											if len(associationForeignKeys) == 0 && len(foreignKeys) == 1 {
												associationForeignKeys = []string{scope.PrimaryKey()}
											}
										} else if len(foreignKeys) != len(associationForeignKeys) {
											scope.Err(errors.New("invalid foreign keys, should have same length"))
											return
										}
									}

									for idx, foreignKey := range foreignKeys {
										if foreignField := getForeignField(foreignKey, toFields); foreignField != nil {
											if associationField := getForeignField(associationForeignKeys[idx], modelStruct.StructFields); associationField != nil {
												// source foreign keys
												foreignField.IsForeignKey = true
												relationship.AssociationForeignFieldNames = append(relationship.AssociationForeignFieldNames, associationField.Name)
												relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, associationField.DBName)

												// association foreign keys
												relationship.ForeignFieldNames = append(relationship.ForeignFieldNames, foreignField.Name)
												relationship.ForeignDBNames = append(relationship.ForeignDBNames, foreignField.DBName)
											}
										}
									}

									if len(relationship.ForeignFieldNames) != 0 {
										field.Relationship = relationship
									}
								}
							} else {
								field.IsNormal = true
							}
						}(field)
					case reflect.Struct:
						defer func(field *StructField) {
							var (
								// user has one profile, associationType is User, profile use UserID as foreign key
								// user belongs to profile, associationType is Profile, user use ProfileID as foreign key
								associationType           = reflectType.Name()
								relationship              = &Relationship{}
								toScope                   = scope.New(reflect.New(field.Struct.Type).Interface())
								toFields                  = toScope.GetStructFields()
								tagForeignKeys            []string
								tagAssociationForeignKeys []string
							)

							if foreignKey := field.TagSettings[TagSettingForeignKey]; foreignKey != "" {
								tagForeignKeys = strings.Split(foreignKey, ",")
							}

							if foreignKey := field.TagSettings[TagSettingAssociationForeignKey]; foreignKey != "" {
								tagAssociationForeignKeys = strings.Split(foreignKey, ",")
							}

							// Has One
							{
								var foreignKeys = tagForeignKeys
								var associationForeignKeys = tagAssociationForeignKeys
								// if no foreign keys defined with tag
								if len(foreignKeys) == 0 {
									// if no association foreign keys defined with tag
									if len(associationForeignKeys) == 0 {
										for _, primaryField := range modelStruct.PrimaryFields {
											foreignKeys = append(foreignKeys, associationType+primaryField.Name)
											associationForeignKeys = append(associationForeignKeys, primaryField.Name)
										}
									} else {
										// generate foreign keys form association foreign keys
										for _, associationForeignKey := range tagAssociationForeignKeys {
											if foreignField := getForeignField(associationForeignKey, modelStruct.StructFields); foreignField != nil {
												foreignKeys = append(foreignKeys, associationType+foreignField.Name)
												associationForeignKeys = append(associationForeignKeys, foreignField.Name)
											}
										}
									}
								} else {
									// generate association foreign keys from foreign keys
									if len(associationForeignKeys) == 0 {
										for _, foreignKey := range foreignKeys {
											if strings.HasPrefix(foreignKey, associationType) {
												associationForeignKey := strings.TrimPrefix(foreignKey, associationType)
												if foreignField := getForeignField(associationForeignKey, modelStruct.StructFields); foreignField != nil {
													associationForeignKeys = append(associationForeignKeys, associationForeignKey)
												}
											}
										}
										if len(associationForeignKeys) == 0 && len(foreignKeys) == 1 {
											associationForeignKeys = []string{scope.PrimaryKey()}
										}
									} else if len(foreignKeys) != len(associationForeignKeys) {
										scope.Err(errors.New("invalid foreign keys, should have same length"))
										return
									}
								}

								for idx, foreignKey := range foreignKeys {
									if foreignField := getForeignField(foreignKey, toFields); foreignField != nil {
										if scopeField := getForeignField(associationForeignKeys[idx], modelStruct.StructFields); scopeField != nil {
											foreignField.IsForeignKey = true
											// source foreign keys
											relationship.AssociationForeignFieldNames = append(relationship.AssociationForeignFieldNames, scopeField.Name)
											relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, scopeField.DBName)

											// association foreign keys
											relationship.ForeignFieldNames = append(relationship.ForeignFieldNames, foreignField.Name)
											relationship.ForeignDBNames = append(relationship.ForeignDBNames, foreignField.DBName)
										}
									}
								}
							}

							if len(relationship.ForeignFieldNames) != 0 {
								relationship.Kind = "has_one"
								field.Relationship = relationship
							} else {
								var foreignKeys = tagForeignKeys
								var associationForeignKeys = tagAssociationForeignKeys

								if len(foreignKeys) == 0 {
									// generate foreign keys & association foreign keys
									if len(associationForeignKeys) == 0 {
										for _, primaryField := range toScope.PrimaryFields() {
											foreignKeys = append(foreignKeys, field.Name+primaryField.Name)
											associationForeignKeys = append(associationForeignKeys, primaryField.Name)
										}
									} else {
										// generate foreign keys with association foreign keys
										for _, associationForeignKey := range associationForeignKeys {
											if foreignField := getForeignField(associationForeignKey, toFields); foreignField != nil {
												foreignKeys = append(foreignKeys, field.Name+foreignField.Name)
												associationForeignKeys = append(associationForeignKeys, foreignField.Name)
											}
										}
									}
								} else {
									// generate foreign keys & association foreign keys
									if len(associationForeignKeys) == 0 {
										for _, foreignKey := range foreignKeys {
											if strings.HasPrefix(foreignKey, field.Name) {
												associationForeignKey := strings.TrimPrefix(foreignKey, field.Name)
												if foreignField := getForeignField(associationForeignKey, toFields); foreignField != nil {
													associationForeignKeys = append(associationForeignKeys, associationForeignKey)
												}
											}
										}
										if len(associationForeignKeys) == 0 && len(foreignKeys) == 1 {
											associationForeignKeys = []string{toScope.PrimaryKey()}
										}
									} else if len(foreignKeys) != len(associationForeignKeys) {
										scope.Err(errors.New("invalid foreign keys, should have same length"))
										return
									}
								}

								for idx, foreignKey := range foreignKeys {
									if foreignField := getForeignField(foreignKey, modelStruct.StructFields); foreignField != nil {
										if associationField := getForeignField(associationForeignKeys[idx], toFields); associationField != nil {
											foreignField.IsForeignKey = true

											// association foreign keys
											relationship.AssociationForeignFieldNames = append(relationship.AssociationForeignFieldNames, associationField.Name)
											relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, associationField.DBName)

											// source foreign keys
											relationship.ForeignFieldNames = append(relationship.ForeignFieldNames, foreignField.Name)
											relationship.ForeignDBNames = append(relationship.ForeignDBNames, foreignField.DBName)
										}
									}
								}

								if len(relationship.ForeignFieldNames) != 0 {
									relationship.Kind = "belongs_to"
									field.Relationship = relationship
								}
							}
						}(field)
					default:
						field.IsNormal = true
					}
				}
			}

			// Even it is ignored, also possible to decode db value into the field
			field.DBName = ToDBName(fieldStruct.Name)

			modelStruct.StructFields = append(modelStruct.StructFields, field)
		}
	}

	if len(modelStruct.PrimaryFields) == 0 {
		if field := getForeignField("id", modelStruct.StructFields); field != nil {
			field.IsPrimaryKey = true
			modelStruct.PrimaryFields = append(modelStruct.PrimaryFields, field)
		}
	}

	modelStructsMap.Set(reflectType, &modelStruct)

	for _, sf := range modelStruct.StructFields {
		modelStruct.StructFieldsByName[sf.Name] = sf
	}

	return &modelStruct
}

// GetStructFields get model's field structs
func (scope *Scope) GetStructFields() (fields []*StructField) {
	return scope.GetModelStruct().StructFields
}

func parseTagSetting(reflectType reflect.Type, fieldName string, tags reflect.StructTag) map[TagSetting]string {
	setting := map[TagSetting]string{}
	if str, ok := tags.Lookup("hades"); ok {
		tags := strings.Split(str, ";")
		for _, value := range tags {
			v := strings.Split(value, ":")
			k := strings.TrimSpace(v[0])

			if _, ok := ValidTagSettings[TagSetting(k)]; !ok {
				var validTags []string
				for vt := range ValidTagSettings {
					validTags = append(validTags, string(vt))
				}
				panic(fmt.Sprintf("invalid tag setting %q for field %s of type %v - valid tag settings are %s",
					k,
					fieldName,
					reflectType,
					strings.Join(validTags, ", "),
				))
			}

			if len(v) >= 2 {
				setting[TagSetting(k)] = strings.Join(v[1:], ":")
			} else {
				setting[TagSetting(k)] = k
			}
		}
	}
	return setting
}

// JoinTableHandler default join table handler
type JoinTableHandler struct {
	TableName   string          `sql:"-"`
	Source      JoinTableSource `sql:"-"`
	Destination JoinTableSource `sql:"-"`
}

// SourceForeignKeys return source foreign keys
func (s *JoinTableHandler) SourceForeignKeys() []JoinTableForeignKey {
	return s.Source.ForeignKeys
}

// DestinationForeignKeys return destination foreign keys
func (s *JoinTableHandler) DestinationForeignKeys() []JoinTableForeignKey {
	return s.Destination.ForeignKeys
}

// Setup initialize a default join table handler
func (s *JoinTableHandler) Setup(relationship *Relationship, tableName string, source reflect.Type, destination reflect.Type) {
	s.TableName = tableName

	s.Source = JoinTableSource{ModelType: source}
	s.Source.ForeignKeys = []JoinTableForeignKey{}
	for idx, dbName := range relationship.ForeignFieldNames {
		s.Source.ForeignKeys = append(s.Source.ForeignKeys, JoinTableForeignKey{
			DBName:            relationship.ForeignDBNames[idx],
			AssociationDBName: dbName,
		})
	}

	s.Destination = JoinTableSource{ModelType: destination}
	s.Destination.ForeignKeys = []JoinTableForeignKey{}
	for idx, dbName := range relationship.AssociationForeignFieldNames {
		s.Destination.ForeignKeys = append(s.Destination.ForeignKeys, JoinTableForeignKey{
			DBName:            relationship.AssociationForeignDBNames[idx],
			AssociationDBName: dbName,
		})
	}
}

// Table return join table's table name
func (s JoinTableHandler) Table() string {
	return s.TableName
}

func init() {
	inflection.AddIrregular("human", "humans")
}

func TableName(typ reflect.Type) string {
	return ToDBName(inflection.Plural(typ.Name()))
}
