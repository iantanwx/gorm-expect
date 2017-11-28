package gormexpect

import (
	"database/sql"
	"database/sql/driver"

	"github.com/jinzhu/gorm"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

var (
	db   *sql.DB
	mock sqlmock.Sqlmock
)

func init() {
	var err error

	db, mock, err = sqlmock.NewWithDSN("mock_gorm_dsn")

	if err != nil {
		panic(err.Error())
	}
}

// Adapter provides an abstract interface over concrete mock database
// implementations (e.g. go-sqlmock or go-testdb)
type Adapter interface {
	ExpectQuery(stmt Stmt) Queryer
	ExpectExec(stmt Stmt) ExpectedExec
	AssertExpectations() error
}

// NewSqlmockAdapter returns a mock gorm.DB and an Adapter backed by
// go-sqlmock
func NewSqlmockAdapter(dialect string, args ...interface{}) (*gorm.DB, Adapter, error) {
	gormDb, err := gorm.Open("sqlmock", "mock_gorm_dsn")

	if err != nil {
		return nil, nil, err
	}

	return gormDb, &SqlmockAdapter{db: db, mocker: mock}, nil
}

// SqlmockAdapter implemenets the Adapter interface using go-sqlmock
// it is the default Adapter
type SqlmockAdapter struct {
	db     *sql.DB
	mocker sqlmock.Sqlmock
}

// ExpectQuery wraps the underlying mock method for setting a query
// expectation. It accepts multiple statements in the event of preloading
func (a *SqlmockAdapter) ExpectQuery(query Stmt) Queryer {
	expectation := a.mocker.ExpectQuery(query.sql)
	return &SqlmockQueryer{query: expectation}
}

// ExpectExec wraps the underlying mock method for setting a exec
// expectation
func (a *SqlmockAdapter) ExpectExec(exec Stmt) ExpectedExec {
	return &SqlmockExec{mock: a.mocker, exec: exec}
}

// Queryer is returned from ExpectQuery
// it is used to control the mock database's response
type Queryer interface {
	Returns(rows interface{}) Queryer
	Errors(err error) Queryer
	Args(args ...driver.Value) Queryer
}

// SqlmockQueryer implements Queryer
type SqlmockQueryer struct {
	query *sqlmock.ExpectedQuery
}

// Returns will set the low level rows to be returned for a given set of
// queries
func (r *SqlmockQueryer) Returns(rows interface{}) Queryer {
	sqlmockRows, ok := rows.(*sqlmock.Rows)

	if !ok {
		panic("Unsupported type passed to Returns")
	}

	expectation := r.query.WillReturnRows(sqlmockRows)
	return &SqlmockQueryer{query: expectation}
}

// Errors will return an error as the query result
func (r *SqlmockQueryer) Errors(err error) Queryer {
	expectation := r.query.WillReturnError(err)
	return &SqlmockQueryer{query: expectation}
}

// Args sets the args that queries should be executed with
func (r *SqlmockQueryer) Args(args ...driver.Value) Queryer {
	expectation := r.query.WithArgs(args...)
	return &SqlmockQueryer{query: expectation}
}

// AssertExpectations asserts that _all_ expectations for a test have been met
// and returns an error specifying which have not if there are unmet
// expectations
func (a *SqlmockAdapter) AssertExpectations() error {
	return a.mocker.ExpectationsWereMet()
}
