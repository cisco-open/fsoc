package uql

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// jsonResult is a representation of the UQL response that could be directly serialized to JSON.
// The format is such that every value is stored in a JSON object field named as the column alias in the response.
// Column order of the response data is fixed by the UQL query. We want the JSON result to honor this guarantee.
// In Go, only structs allow control over the order of JSON fields. For that reason, we dynamically generate
// anonymous struct types with fields based on the model of the response data.
type jsonResult struct {
	Model any   `json:"model"`
	Data  []any `json:"data"`
}

// valueExtractor transforms value from UQL Response to something serializable as a part of jsonResult.
type valueExtractor func(any) any

// valueSetter assigns extracted value by valueExtractor to a struct field using reflection.
type valueSetter func(reflect.Value, any)

// fieldMapper contains transformation logic for single column of the UQL response
type fieldMapper struct {
	valueExtractor valueExtractor
	valueSetter    valueSetter
	structField    reflect.StructField
}

// tableMapper contains transformation of a table of data into array of structs.
type tableMapper struct {
	arrayType     reflect.Type
	rowType       reflect.Type
	columnMappers []fieldMapper
}

// wrappedScalar is a helper type for JSON serialization of dynamically typed scalar values.
type wrappedScalar struct {
	Value any
}

func (s wrappedScalar) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Value)
}

func (s wrappedScalar) MarshalYAML() (interface{}, error) {
	node := yaml.Node{}
	switch s.Value.(type) {
	// We can't let yaml.Node.Encode try to encode strings, because it treats them as a whole documents.
	// We are only considering scalars. If a string, for example, starts with a tab, the Encode will fail.
	case string:
		node = yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: s.Value.(string),
			Tag:   "!!str",
		}
	default:
		err := node.Encode(s.Value)
		if err != nil {
			return nil, err
		}
	}
	return node, nil
}

type jsonDataType int

const (
	integer jsonDataType = iota
	double
	boolean
	stringType
	objectType
	complexArray
)

func (t jsonDataType) name() string {
	switch t {
	case integer, double:
		return "number"
	case boolean:
		return "boolean"
	case stringType:
		return "string"
	case objectType:
		return "undefined"
	default:
		return "string"
	}
}

func jsonTypeForUqlType(uqlType string) jsonDataType {
	switch strings.ToLower(uqlType) {
	case "number", "long":
		return integer
	case "double":
		return double
	case "boolean":
		return boolean
	case "string", "timestamp", "duration", "csv":
		return stringType
	case "object", "json":
		return objectType
	default:
		return complexArray
	}
}

// transformForJsonOutput produces jsonResult from UqlResponse.
func transformForJsonOutput(response *Response) (jsonResult, error) {
	mainModel := response.Model()
	if err := checkAliasCollisions(mainModel); err != nil {
		return jsonResult{}, err
	}
	columnMappers := make([]fieldMapper, len(mainModel.Fields))
	for i, field := range mainModel.Fields {
		columnMappers[i] = makeFieldMapper(field)
	}
	model, _ := transformModel(mainModel.Fields)
	arrayMapping := makeArrayMapper("_", columnMappers)
	dataArray := arrayMapping.valueExtractor(response.Main()).([]any)
	return jsonResult{
		Model: model,
		Data:  dataArray,
	}, nil
}

// checkAliasCollisions checks for duplicate column aliases in a single table.
// UQL allows having tables with columns sharing names. Such table is not serializable to a JSON where object fields
// are named by these column names. Returns an error containing path to the colliding names.
func checkAliasCollisions(model *Model) error {
	collisionPath := findCollision(model.Fields, make([]string, 0))
	if collisionPath != nil {
		return errors.New(
			"Cannot serialize query response to json format. " +
				"There are multiple columns with the same same. " +
				"Values for such columns cannot be represented as a single JSON object field. " +
				"Please, provide an explicit alias in the query for the value on path in the response structure: " +
				strings.Join(collisionPath, " -> "),
		)
	}
	return nil
}

func findCollision(fields []ModelField, path []string) []string {

	seen := make(map[string]bool, len(fields))
	for _, field := range fields {
		alias := field.Alias
		if seen[alias] {
			return append(path, alias)
		}
		seen[alias] = true
		if field.Model != nil {
			if deeperPath := findCollision(field.Model.Fields, append(path, alias)); deeperPath != nil {
				return deeperPath
			}
		}
	}
	return nil
}

// transformModel creates an anonymous struct where each field is named by a column of the UQL response
// and each value corresponds with the expected serialized JSON data type.
func transformModel(fields []ModelField) (any, reflect.Type) {
	structFields := make([]reflect.StructField, len(fields))
	values := make([]any, len(fields))
	for i, field := range fields {
		tag := serializationTags(field.Alias)
		var jsonType reflect.Type
		if field.Model != nil {
			model, typ := transformModel(field.Model.Fields)
			jsonType = typ
			values[i] = model
		} else {
			model, typ := transformModelField(field)
			jsonType = typ
			values[i] = model
		}
		structField := reflect.StructField{
			Name: fieldIdentifier(field.Alias),
			Type: jsonType,
			Tag:  reflect.StructTag(tag),
		}
		structFields[i] = structField
	}

	modelStructType := reflect.StructOf(structFields)
	reflected := reflect.New(modelStructType).Elem()
	for i, structField := range structFields {
		fieldValue := reflected.FieldByName(structField.Name)
		fieldValue.Set(reflect.ValueOf(values[i]))
	}
	return reflected.Interface(), modelStructType
}

// transformModelField creates single field and value for the whole response model transformation.
func transformModelField(field ModelField) (any, reflect.Type) {
	if field.Model != nil {
		return transformModel(field.Model.Fields)
	} else {
		return jsonTypeForUqlType(field.Type).name(), reflect.TypeOf(field.Alias)
	}
}

func makeFieldMapper(field ModelField) fieldMapper {
	if field.Model != nil {
		return makeStructArrayFieldMapper(field)
	} else {
		return makeSimpleFieldMapper(field)
	}
}

func makeSimpleFieldMapper(model ModelField) fieldMapper {
	tag := serializationTags(model.Alias)
	structField := reflect.StructField{
		Name: fieldIdentifier(model.Alias),
		Type: reflect.TypeOf(wrappedScalar{}),
		Tag:  reflect.StructTag(tag),
	}
	return fieldMapper{
		valueExtractor: wrapIdentity,
		valueSetter:    wrappedScalarSetter,
		structField:    structField,
	}
}

func makeStructArrayFieldMapper(model ModelField) fieldMapper {
	fields := make([]fieldMapper, len(model.Model.Fields))

	for i, field := range model.Model.Fields {
		fields[i] = makeFieldMapper(field)
	}

	return makeArrayMapper(model.Alias, fields)
}

func makeArrayMapper(alias string, fieldMappers []fieldMapper) fieldMapper {

	structFields := make([]reflect.StructField, len(fieldMappers))
	for i, field := range fieldMappers {
		structFields[i] = field.structField
	}

	rowType := reflect.StructOf(structFields)
	arrayType := reflect.SliceOf(rowType)
	tableMapper := tableMapper{
		arrayType:     arrayType,
		rowType:       rowType,
		columnMappers: fieldMappers,
	}
	tag := serializationTags(alias)
	arrayField := reflect.StructField{
		Name: fieldIdentifier(alias),
		Type: arrayType,
		Tag:  reflect.StructTag(tag),
	}
	return fieldMapper{
		valueExtractor: tableExtractor(tableMapper),
		valueSetter:    arraySetter,
		structField:    arrayField,
	}
}

func wrapIdentity(value any) any {
	return wrappedScalar{
		Value: value,
	}
}

func tableExtractor(mapper tableMapper) valueExtractor {
	return func(data any) any {
		if cast, ok := data.(Complex); ok {
			return mapToTable(cast, mapper)
		}
		panic(fmt.Sprintf("Cannot map values to a table. Received an unexpected type %s", reflect.TypeOf(data)))
	}
}

func mapToTable(dataset Complex, tableMapper tableMapper) []any {
	if complexIsNil(dataset) {
		return nil
	}
	rows := make([]any, len(dataset.Values()))
	for rowIdx, row := range dataset.Values() {
		rowStruct := reflect.New(tableMapper.rowType).Elem()
		for colIdx, colVal := range row {
			mapper := tableMapper.columnMappers[colIdx]
			structField := rowStruct.FieldByName(mapper.structField.Name)
			mapper.valueSetter(structField, mapper.valueExtractor(colVal))
		}
		rows[rowIdx] = rowStruct.Interface()
	}
	return rows
}

// value and receiver type must be of slice type
func arraySetter(receiver reflect.Value, value any) {
	cast := value.([]interface{})
	if cast != nil && len(cast) == 0 {
		receiver.Set(reflect.MakeSlice(receiver.Type(), 0, 0))
	} else {
		asValues := make([]reflect.Value, len(cast))
		for i, item := range cast {
			asValues[i] = reflect.ValueOf(item)
		}
		receiver.Set(reflect.Append(receiver, asValues...))
	}

}

// value and receiver type must be of type wrappedScalar
func wrappedScalarSetter(receiver reflect.Value, value any) {
	receiver.Set(reflect.ValueOf(value))
}

func fieldIdentifier(alias string) string {
	return "Exported_" + hex.EncodeToString([]byte(alias))
}

func serializationTags(alias string) string {
	return fmt.Sprintf("json:\"%s\" yaml:\"%s\"", alias, alias)
}
