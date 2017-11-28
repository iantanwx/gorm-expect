package gormexpect

import (
	"database/sql/driver"
	"reflect"

	"github.com/jinzhu/gorm"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

// ExpectedExec represents an expected exec that will be executed and can
// return a result. It presents a fluent API for chaining calls to other
// expectations
type ExpectedExec interface {
	WillSucceed(lastInsertID, rowsAffected int64) ExpectedExec
	WillFail(err error) ExpectedExec
}

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

// SqlmockExec implements Exec for go-sqlmock
type SqlmockExec struct {
	exec Stmt
	mock sqlmock.Sqlmock
}

// WillSucceed accepts a two int64s. They are passed directly to the underlying
// mock db. Useful for checking DAO behaviour in the event that the incorrect
// number of rows are affected by an Exec
func (e *SqlmockExec) WillSucceed(lastReturnedID, rowsAffected int64) ExpectedExec {
	result := sqlmock.NewResult(lastReturnedID, rowsAffected)
	e.mock.ExpectExec(e.exec.sql).WillReturnResult(result)

	return e
}

// WillFail simulates returning an Error from an unsuccessful exec
func (e *SqlmockExec) WillFail(err error) ExpectedExec {
	result := sqlmock.NewErrorResult(err)
	e.mock.ExpectExec(e.exec.sql).WillReturnResult(result)

	return e
}
