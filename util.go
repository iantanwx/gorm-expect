package gormexpect

import (
	"database/sql/driver"
	"reflect"
	"regexp"
	"strings"
	"unsafe"

	"github.com/jinzhu/gorm"
)

func parseUpdateColumns(stmt string) ([]string, []string) {
	re := regexp.MustCompile(`SET (.*)( ?:WHERE)?`)

	match := re.FindAllStringSubmatch(stmt, -1)

	if len(match) == 0 {
		return nil, nil
	}

	index := re.FindAllStringSubmatchIndex(stmt, -1)

	before := stmt[0:index[0][2]]
	after := stmt[index[0][3]:]

	colsString := match[0][1]
	cols := strings.Split(colsString, ",")

	for i, col := range cols {
		cols[i] = strings.TrimSpace(col)
	}

	return []string{before, after}, cols
}

// indirect returns the actual value if the given value is a pointer
func indirect(reflectValue reflect.Value) reflect.Value {
	for reflectValue.Kind() == reflect.Ptr {
		reflectValue = reflectValue.Elem()
	}
	return reflectValue
}

// Preload mirrors gorm's search.searchPreload
// since it's private, we have to resort to some reflection black magic to
// make it work right. we'll just read from private field using reflect and
// copy the values into Preload.
type Preload struct {
	schema     string
	conditions []interface{}
}

// getPreload copies preload from scope.Search, because it is a private field
// and therefore inaccessible to this package by normal methods
func getPreload(scope *gorm.Scope) []Preload {
	var preload []Preload
	searchVal := indirect(reflect.ValueOf(scope.Search))
	preloadVal := searchVal.FieldByName("preload")

	if preloadVal.Kind() == reflect.Slice && preloadVal.Len() > 0 {
		for i := 0; i < preloadVal.Len(); i++ {
			elem := preloadVal.Index(i)
			schemaVal := elem.FieldByName("schema")
			schemaVal = reflect.NewAt(schemaVal.Type(), unsafe.Pointer(schemaVal.UnsafeAddr())).Elem()
			schema := (schemaVal.Interface()).(string)
			conditionsVal := elem.FieldByName("conditions")
			conditionsVal = reflect.NewAt(conditionsVal.Type(), unsafe.Pointer(conditionsVal.UnsafeAddr())).Elem()
			conditions := (conditionsVal.Interface()).([]interface{})

			preloadElem := Preload{schema, conditions}
			preload = append(preload, preloadElem)
		}
	}

	return preload
}

// getRowForFields accepts a gorm.Field and converts them to []driver.Value so
// that they can then be turned into sql.Rows
func getRowForFields(fields []*gorm.Field) []driver.Value {
	var values []driver.Value
	for _, field := range fields {
		if field.IsNormal {
			value := field.Field

			// dereference pointers
			if field.Field.Kind() == reflect.Ptr {
				value = reflect.Indirect(field.Field)
			}

			// check if we have a zero Value
			// just append nil if it's not valid, so sqlmock won't complain
			if !value.IsValid() {
				values = append(values, nil)
				continue
			}

			concreteVal := value.Interface()

			if driver.IsValue(concreteVal) {
				values = append(values, concreteVal)
			} else if num, err := driver.DefaultParameterConverter.ConvertValue(concreteVal); err == nil {
				values = append(values, num)
			} else if valuer, ok := concreteVal.(driver.Valuer); ok {
				if convertedValue, err := valuer.Value(); err == nil {
					values = append(values, convertedValue)
				}
			}
		}
	}

	return values
}
