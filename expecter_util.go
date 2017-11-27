package gormexpect

import (
	"reflect"
	"unsafe"

	"github.com/jinzhu/gorm"
)

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
