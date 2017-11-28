package gormexpect

import (
	"fmt"
	"reflect"

	"github.com/jinzhu/gorm"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

// QueryExpectation is returned by Expecter. It exposes a narrower API than
// Queryer to limit footguns.
type QueryExpectation interface {
	Returns(value interface{}) QueryExpectation
	// Error(err error) QueryExpectation
}

// SqlmockQueryExpectation implements QueryExpectation for go-sqlmock
// It gets a pointer to Expecter
type SqlmockQueryExpectation struct {
	parent *Expecter
	scope  *gorm.Scope
}

// Returns accepts an out type which should either be a struct or slice. Under
// the hood, it converts a gorm model struct to sql.Rows that can be passed to
// the underlying mock db
func (q *SqlmockQueryExpectation) Returns(out interface{}) QueryExpectation {
	scope := (&gorm.Scope{}).New(out)
	q.scope = scope
	// call deferred queries, since we now know the expected out value
	q.callMethods()

	outVal := indirect(reflect.ValueOf(out))

	destQuery := q.parent.recorder.stmts[0]
	subQueries := q.parent.recorder.stmts[1:]

	// main query always at the head of the slice
	q.parent.adapter.ExpectQuery(destQuery).Returns(q.getDestRows(out))

	// subqueries are preload
	for _, subQuery := range subQueries {
		if subQuery.preload != "" {
			if field, ok := scope.FieldByName(subQuery.preload); ok {
				expectation := q.parent.adapter.ExpectQuery(subQuery)
				rows, hasRows := q.getRelationRows(outVal.FieldByName(subQuery.preload), subQuery.preload, field.Relationship)

				if hasRows {
					expectation.Returns(rows)
				}
			}
		}
	}

	return q
}

func (q *SqlmockQueryExpectation) getRelationRows(rVal reflect.Value, fieldName string, relation *gorm.Relationship) (*sqlmock.Rows, bool) {
	var (
		rows    *sqlmock.Rows
		columns []string
	)

	// we need to check for zero values
	if reflect.DeepEqual(rVal.Interface(), reflect.New(rVal.Type()).Elem().Interface()) {
		return nil, false
	}

	switch relation.Kind {
	case "has_one":
		scope := &gorm.Scope{Value: rVal.Interface()}

		for _, field := range scope.GetModelStruct().StructFields {
			if field.IsNormal {
				columns = append(columns, field.DBName)
			}
		}

		rows = sqlmock.NewRows(columns)

		// we don't have a slice
		row := getRowForFields(scope.Fields())
		rows = rows.AddRow(row...)

		return rows, true
	case "has_many":
		elem := rVal.Type().Elem()
		scope := &gorm.Scope{Value: reflect.New(elem).Interface()}

		for _, field := range scope.GetModelStruct().StructFields {
			if field.IsNormal {
				columns = append(columns, field.DBName)
			}
		}

		rows = sqlmock.NewRows(columns)

		if rVal.Len() > 0 {
			for i := 0; i < rVal.Len(); i++ {
				scope := &gorm.Scope{Value: rVal.Index(i).Interface()}
				row := getRowForFields(scope.Fields())
				rows = rows.AddRow(row...)
			}

			return rows, true
		}

		return nil, false
	case "many_to_many":
		elem := rVal.Type().Elem()
		scope := &gorm.Scope{Value: reflect.New(elem).Interface()}
		joinTable := relation.JoinTableHandler.(*gorm.JoinTableHandler)

		for _, field := range scope.GetModelStruct().StructFields {
			if field.IsNormal {
				columns = append(columns, field.DBName)
			}
		}

		for _, key := range joinTable.Source.ForeignKeys {
			columns = append(columns, key.DBName)
		}

		for _, key := range joinTable.Destination.ForeignKeys {
			columns = append(columns, key.DBName)
		}

		rows = sqlmock.NewRows(columns)

		// in this case we definitely have a slice
		if rVal.Len() > 0 {
			for i := 0; i < rVal.Len(); i++ {
				scope := &gorm.Scope{Value: rVal.Index(i).Interface()}
				row := getRowForFields(scope.Fields())

				// need to append the values for join table keys
				sourcePk := q.scope.PrimaryKeyValue()
				destModelType := joinTable.Destination.ModelType
				destModelVal := reflect.New(destModelType).Interface()
				destPkVal := (&gorm.Scope{Value: destModelVal}).PrimaryKeyValue()

				row = append(row, sourcePk, destPkVal)

				rows = rows.AddRow(row...)
			}

			return rows, true
		}

		return nil, false
	default:
		return nil, false
	}
}

func (q *SqlmockQueryExpectation) getDestRows(out interface{}) *sqlmock.Rows {
	var columns []string
	for _, field := range (&gorm.Scope{}).New(out).GetModelStruct().StructFields {
		if field.IsNormal {
			columns = append(columns, field.DBName)
		}
	}

	rows := sqlmock.NewRows(columns)
	outVal := indirect(reflect.ValueOf(out))

	// SELECT multiple columns
	if outVal.Kind() == reflect.Slice {
		outSlice := []interface{}{}

		for i := 0; i < outVal.Len(); i++ {
			outSlice = append(outSlice, outVal.Index(i).Interface())
		}

		for _, outElem := range outSlice {
			scope := &gorm.Scope{Value: outElem}
			row := getRowForFields(scope.Fields())
			rows = rows.AddRow(row...)
		}
	} else if outVal.Kind() == reflect.Struct { // SELECT with LIMIT 1
		row := getRowForFields(q.scope.Fields())
		rows = rows.AddRow(row...)
	} else {
		panic(fmt.Errorf("Can only get rows for slice or struct"))
	}

	return rows
}

// callMethods is used to call deferred db.* methods. It is necessary to
// ensure scope.Value has a primary key, extracted from the model passed to
// SqlmockQueryExpectation.Returns. This is because the noop database does not
// return any actual rows.
func (q *SqlmockQueryExpectation) callMethods() {
	q.parent.gorm = q.parent.gorm.Set("gorm_expect:ret", q.scope.Value)
	q.parent.gorm.Callback().Query().Before("gorm:preload").Register("gorm_expect:populate_scope_val", populateScopeValueCallback)

	noop := reflect.ValueOf(q.parent.gorm)
	for methodName, args := range q.parent.callmap {
		methodVal := noop.MethodByName(methodName)

		switch method := methodVal.Interface().(type) {
		case func(interface{}, ...interface{}) *gorm.DB:
			method(args[0], args[1:]...)
		default:
			fmt.Println("Not a supported method signature")
		}
	}
}
