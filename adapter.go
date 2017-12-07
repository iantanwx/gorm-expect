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
	ExpectExec(stmt Stmt) Execer
	ExpectBegin() TxBeginner
	ExpectCommit() TxCommitter
	ExpectRollback() TxRollback
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
func (a *SqlmockAdapter) ExpectExec(exec Stmt) Execer {
	expectation := a.mocker.ExpectExec(exec.sql)
	return &SqlmockExecer{exec: expectation}
}

// ExpectBegin mocks a sql transaction
func (a *SqlmockAdapter) ExpectBegin() TxBeginner {
	expectation := a.mocker.ExpectBegin()
	return &SqlmockTxBeginner{begin: expectation}
}

// ExpectCommit mocks committing a sql transaction
func (a *SqlmockAdapter) ExpectCommit() TxCommitter {
	expectation := a.mocker.ExpectCommit()

	return &SqlmockTxCommitter{commit: expectation}
}

// ExpectRollback mocks rolling back a sql
func (a *SqlmockAdapter) ExpectRollback() TxRollback {
	expectation := a.mocker.ExpectRollback()

	return &SqlmockTxRollback{rollback: expectation}
}

// AssertExpectations asserts that _all_ expectations for a test have been met
// and returns an error specifying which have not if there are unmet
// expectations
func (a *SqlmockAdapter) AssertExpectations() error {
	return a.mocker.ExpectationsWereMet()
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

// Execer is a high-level interface to the underlying mock db
type Execer interface {
	WillSucceed(lastInsertID, rowsAffected int64) Execer
	WillFail(err error) Execer
	Args(args ...driver.Value) Execer
}

// SqlmockExecer implements Execer with gosqlmock
type SqlmockExecer struct {
	exec *sqlmock.ExpectedExec
}

// WillSucceed accepts a two int64s. They are passed directly to the underlying
// mock db. Useful for checking DAO behaviour in the event that the incorrect
// number of rows are affected by an Exec
func (e *SqlmockExecer) WillSucceed(lastReturnedID, rowsAffected int64) Execer {
	result := sqlmock.NewResult(lastReturnedID, rowsAffected)
	expectation := e.exec.WillReturnResult(result)

	return &SqlmockExecer{exec: expectation}
}

// WillFail simulates returning an Error from an unsuccessful exec
func (e *SqlmockExecer) WillFail(err error) Execer {
	expectation := e.exec.WillReturnError(err)

	return &SqlmockExecer{exec: expectation}
}

// Args sets the args that the statement should be executed with
func (e *SqlmockExecer) Args(args ...driver.Value) Execer {
	expectation := e.exec.WithArgs(args...)
	return &SqlmockExecer{exec: expectation}
}

// TxBeginner is an interface to underlying sql.Driver mock implementation
type TxBeginner interface {
	WillFail(err error) TxBeginner
}

// SqlmockTxBeginner implements TxBeginner
type SqlmockTxBeginner struct {
	begin *sqlmock.ExpectedBegin
}

// WillFail implements TxBeginner
func (b *SqlmockTxBeginner) WillFail(err error) TxBeginner {
	expectation := b.begin.WillReturnError(err)
	return &SqlmockTxBeginner{begin: expectation}
}

// TxRollback is an interface to underlying mock implementation's tx.Rollback
type TxRollback interface {
	WillFail(err error) TxRollback
}

// SqlmockTxCloser implement TxCloser
type SqlmockTxRollback struct {
	rollback *sqlmock.ExpectedRollback
}

// WillFail implements TxCloser
func (c *SqlmockTxRollback) WillFail(err error) TxRollback {
	expectation := c.rollback.WillReturnError(err)
	return &SqlmockTxRollback{rollback: expectation}
}

// TxCommitter is an interface to underlying mock implementation's tx.Commit
type TxCommitter interface {
	WillFail(err error) TxCommitter
}

// SqlmockTxCommitter implements TxCommitter
type SqlmockTxCommitter struct {
	commit *sqlmock.ExpectedCommit
}

// WillFail implements TxCommitter
func (c *SqlmockTxCommitter) WillFail(err error) TxCommitter {
	expectation := c.commit.WillReturnError(err)
	return &SqlmockTxCommitter{commit: expectation}
}
