package gormexpect

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"sync"

	"github.com/jinzhu/gorm"
)

var pool *NoopDriver

func init() {
	pool = &NoopDriver{
		conns: make(map[string]*NoopConnection),
	}

	sql.Register("noop", pool)
}

// NoopDriver implements sql/driver.Driver
type NoopDriver struct {
	sync.Mutex
	counter int
	conns   map[string]*NoopConnection
}

// Open implements sql/driver.Driver
func (d *NoopDriver) Open(dsn string) (driver.Conn, error) {
	d.Lock()
	defer d.Unlock()

	c, ok := d.conns[dsn]

	if !ok {
		return c, fmt.Errorf("No connection available")
	}

	c.opened++
	return c, nil
}

// NoopResult is a noop struct that satisfies sql.Result
type NoopResult struct {
	lastInsertID int64
	rowsAffected int64
}

// LastInsertId is a noop method for satisfying drive.Result
func (r NoopResult) LastInsertId() (int64, error) {
	return r.lastInsertID, nil
}

// RowsAffected is a noop method for satisfying drive.Result
func (r NoopResult) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}

// NoopRows implements driver.Rows
type NoopRows struct {
	pos int
}

// Columns implements driver.Rows
func (r *NoopRows) Columns() []string {
	return []string{"foo", "bar", "baz", "lol", "kek", "zzz"}
}

// Close implements driver.Rows
func (r *NoopRows) Close() error {
	return nil
}

// Next implements driver.Rows and alwys returns only one row
func (r *NoopRows) Next(dest []driver.Value) error {
	if r.pos == 1 {
		return io.EOF
	}
	cols := []string{"foo", "bar", "baz", "lol", "kek", "zzz"}

	for i, col := range cols {
		dest[i] = col
	}

	r.pos++

	return nil
}

// NoopStmt implements driver.Stmt
type NoopStmt struct{}

// Close implements driver.Stmt
func (s *NoopStmt) Close() error {
	return nil
}

// NumInput implements driver.Stmt
func (s *NoopStmt) NumInput() int {
	return 1
}

// Exec implements driver.Stmt
func (s *NoopStmt) Exec(args []driver.Value) (driver.Result, error) {
	return &NoopResult{}, nil
}

// Query implements driver.Stmt
func (s *NoopStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &NoopRows{}, nil
}

// NewNoopDB initialises a new DefaultNoopDB
func NewNoopDB() (gorm.SQLCommon, NoopController, error) {
	pool.Lock()
	dsn := fmt.Sprintf("noop_db_%d", pool.counter)
	pool.counter++

	noop := &NoopConnection{nextExecResult: []int64{0, 0}, dsn: dsn, drv: pool}
	pool.conns[dsn] = noop
	pool.Unlock()

	db, err := noop.open()

	return db, noop, err
}

// NoopConnection implements sql/driver.Conn
// for our purposes, the noop connection never returns an error, as we only
// require it for generating queries. It is necessary because eager loading
// will fail if any operation returns an error
type NoopConnection struct {
	dsn            string
	drv            *NoopDriver
	opened         int
	returnNilRows  bool
	nextExecResult []int64
}

func (c *NoopConnection) open() (*sql.DB, error) {
	db, err := sql.Open("noop", c.dsn)

	if err != nil {
		return db, err
	}

	return db, db.Ping()
}

// Close implements sql/driver.Conn
func (c *NoopConnection) Close() error {
	c.drv.Lock()
	defer c.drv.Unlock()

	c.opened--
	if c.opened == 0 {
		delete(c.drv.conns, c.dsn)
	}

	return nil
}

// NoopController provides a crude interface for manipulating NoopConnection
type NoopController interface {
	ReturnNilRows()
	ReturnExecResult(lastReturnedID, rowsAffected int64)
}

// Begin implements sql/driver.Conn
func (c *NoopConnection) Begin() (driver.Tx, error) {
	return c, nil
}

// Exec implements sql/driver.Conn
func (c *NoopConnection) Exec(query string, args []driver.Value) (driver.Result, error) {
	defer func() {
		c.nextExecResult = []int64{0, 0}
	}()

	return NoopResult{c.nextExecResult[0], c.nextExecResult[1]}, nil
}

// Prepare implements sql/driver.Conn
func (c *NoopConnection) Prepare(query string) (driver.Stmt, error) {
	return &NoopStmt{}, nil
}

// Query implements sql/driver.Conn
func (c *NoopConnection) Query(query string, args []driver.Value) (driver.Rows, error) {
	if c.returnNilRows {
		c.returnNilRows = false
		return &NoopRows{pos: 1}, nil
	}

	return &NoopRows{}, nil
}

// ReturnNilRows instructs the noop driver to return empty rows for all queries
// until returnNilRows is set to false
func (c *NoopConnection) ReturnNilRows() {
	c.returnNilRows = true
}

// ReturnExecResult will cause the driver to return the passed values for the
// next call to Exec. It goes back to the default of 0, 0 thereafter.
func (c *NoopConnection) ReturnExecResult(lastReturnedID, rowsAffected int64) {
	c.nextExecResult = []int64{lastReturnedID, rowsAffected}
}

// Commit implements sql/driver.Conn
func (c *NoopConnection) Commit() error {
	return nil
}

// Rollback implements sql/driver.Conn
func (c *NoopConnection) Rollback() error {
	return nil
}
