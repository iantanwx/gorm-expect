package gormexpect

import (
	"fmt"
	"regexp"

	"github.com/jinzhu/gorm"
)

// Recorder satisfies the logger interface
type Recorder struct {
	stmts   []Stmt
	preload []Preload // store it on Recorder
}

// Record records a Stmt for use when SQL is finally executed
func (r *Recorder) Record(stmt Stmt) {
	stmt.sql = regexp.QuoteMeta(stmt.sql)
	r.stmts = append(r.stmts, stmt)
}

// GetFirst returns the first recorded sql statement logged. If there are no
// statements, false is returned
func (r *Recorder) GetFirst() (Stmt, bool) {
	if len(r.stmts) > 0 {
		return r.stmts[0], true
	}

	return Stmt{}, false
}

// IsEmpty returns true if the statements slice is empty
func (r *Recorder) IsEmpty() bool {
	return len(r.stmts) == 0
}

// Stmt represents a sql statement. It can be an Exec, Query, or QueryRow
type Stmt struct {
	kind    string // can be Query, Exec, QueryRow
	preload string // contains schema if it is a preload query
	sql     string
	args    []interface{}
}

func recordExecCallback(scope *gorm.Scope) {
	r, ok := scope.Get("gorm:recorder")

	if !ok {
		panic(fmt.Errorf("Expected a recorder to be set, but got none"))
	}

	stmt := Stmt{
		kind: "exec",
		sql:  scope.SQL,
		args: scope.SQLVars,
	}

	recorder := r.(*Recorder)

	recorder.Record(stmt)
}

func populateScopeValueCallback(scope *gorm.Scope) {
	// we need to see if we have a valid outval
	returnValue, ok := scope.Get("gorm_expect:ret")

	if ok {
		scope.Value = returnValue
	}
}

func recordQueryCallback(scope *gorm.Scope) {
	r, ok := scope.Get("gorm:recorder")

	if !ok {
		panic(fmt.Errorf("Expected a recorder to be set, but got none"))
	}

	recorder := r.(*Recorder)

	stmt := Stmt{
		kind: "query",
		sql:  scope.SQL,
		args: scope.SQLVars,
	}

	if len(recorder.preload) > 0 {
		stmt.preload = recorder.preload[0].schema

		// we just want to pop the first element off
		recorder.preload = recorder.preload[1:]
	}

	recorder.Record(stmt)
}

func recordPreloadCallback(scope *gorm.Scope) {
	// this callback runs _before_ gorm:preload
	recorder, ok := scope.Get("gorm:recorder")

	if !ok {
		panic(fmt.Errorf("Expected a recorder to be set, but got none"))
	}

	preload := getPreload(scope)

	if len(preload) > 0 {
		// spew.Printf("callback:preload\r\n%s\r\n", spew.Sdump(scope.Search.preload))
		recorder.(*Recorder).preload = preload
	}
}
