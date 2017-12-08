package gormexpect

import (
	"bytes"
	"fmt"
	"regexp"

	"github.com/jinzhu/gorm"
)

// Recorder satisfies the logger interface
type Recorder struct {
	blankColumns []string
	stmts        []Stmt
	preload      []Preload // store it on Recorder
}

// Record records a Stmt for use when SQL is finally executed
// By default, it escapes with regexp.EscapeMeta
func (r *Recorder) Record(stmt Stmt, shouldEscape bool) {
	if shouldEscape {
		stmt.sql = regexp.QuoteMeta(stmt.sql)
	}

	r.stmts = append(r.stmts, stmt)
}

// GetFirst returns the first recorded sql statement logged. If there are no
// statements, false is returned
func (r *Recorder) GetFirst() (Stmt, bool) {
	defer func() {
		if len(r.stmts) > 1 {
			r.stmts = r.stmts[1:]
			return
		}

		r.stmts = []Stmt{}
	}()

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
	recorder := r.(*Recorder)

	if !ok {
		panic(fmt.Errorf("Expected a recorder to be set, but got none"))
	}

	if scope.SQL == "" {
		return
	}

	stmt := Stmt{
		kind: "exec",
		sql:  scope.SQL,
		args: scope.SQLVars,
	}

	if blankColumns, ok := scope.InstanceGet("gorm:blank_columns_with_default_value"); ok {
		// use this hack to retrieve our columns later
		recorder.blankColumns = blankColumns.([]string)
	}

	strs, cols := parseUpdateColumns(stmt.sql)

	if len(cols) > 1 {
		// we generate a better regex
		newRegexp := bytes.NewBufferString("")
		newRegexp.WriteString(strs[0])
		newRegexp.WriteString("(")

		for i, col := range cols {
			if i == 0 {
				newRegexp.WriteString(fmt.Sprintf("%s,?|", col))
				continue
			}

			if i == len(cols)-1 {
				newRegexp.WriteString(fmt.Sprintf(" %s,?)*", col))
				continue
			}

			newRegexp.WriteString(fmt.Sprintf(" %s,?|", col))
		}

		newRegexp.WriteString(strs[1])

		stmt.sql = newRegexp.String()

		recorder.Record(stmt, false)
		return
	}

	recorder.Record(stmt, true)
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

	recorder.Record(stmt, true)
}

func recordPreloadCallback(scope *gorm.Scope) {
	// this callback runs _before_ gorm:preload
	recorder, ok := scope.Get("gorm:recorder")

	if !ok {
		panic(fmt.Errorf("Expected a recorder to be set, but got none"))
	}

	preload := getPreload(scope)

	if len(preload) > 0 {
		recorder.(*Recorder).preload = preload
	}
}
