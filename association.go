package gormexpect

import (
	"fmt"

	"github.com/jinzhu/gorm"
)

// Operation represents the Association method being called. Behaviour
// internally changes depending on the operation.
type Operation int

const (
	Find Operation = iota
	Append
	Replace
	Delete
	Count
	Clear
)

// MockAssociation mirros gorm.Association
type MockAssociation struct {
	column          string
	parent          *Expecter
	noopAssociation *gorm.Association
	operation       Operation
}

// QueryWrapper is just a wrapper over QueryExpectation. This is necessary to
// allow MockAssociation to have a fluent API
type QueryWrapper struct {
	association *MockAssociation
	expectation QueryExpectation
}

// Returns functions in the same way as Expecter.Returns
func (w *QueryWrapper) Returns(value interface{}) *MockAssociation {
	w.expectation.Returns(value)
	return w.association
}

// ExecWrapper wraps ExecExpectation
type ExecWrapper struct {
	association *MockAssociation
	expectation ExecExpectation
}

// WillSucceed has the same signature as ExecExpectation.WillSucceed. It is
// only returned from Append() and Replace().
func (w *ExecWrapper) WillSucceed(lastReturnID, rowsAffected int64) {
	switch w.association.operation {
	case Replace:
		handleReplace(w.association, true)
	case Append, Clear, Delete:
		handleAssociationGeneric(w.association)
	default:
		return
	}
}

// NewMockAssociation returns a MockAssociation
func NewMockAssociation(c string, a *gorm.Association, e *Expecter) *MockAssociation {
	return &MockAssociation{column: c, parent: e, noopAssociation: a}
}

// Find wraps gorm.Association
func (a *MockAssociation) Find(value interface{}) *QueryWrapper {
	a.noopAssociation.Find(&value)
	expectation := &SqlmockQueryExpectation{association: a, parent: a.parent}

	return &QueryWrapper{association: a, expectation: expectation}
}

// Append wraps gorm.Association.Append
func (a *MockAssociation) Append(values ...interface{}) *ExecWrapper {
	a.operation = Append
	a.noopAssociation.Append(values...)
	expectation := &SqlmockExecExpectation{parent: a.parent}

	return &ExecWrapper{association: a, expectation: expectation}
}

// Delete wraps gorm.Association.Delete
func (a *MockAssociation) Delete(values ...interface{}) *ExecWrapper {
	a.operation = Delete
	a.noopAssociation.Delete(values...)
	expectation := &SqlmockExecExpectation{parent: a.parent}

	return &ExecWrapper{association: a, expectation: expectation}
}

// Clear wraps gorm.Association.Clear
func (a *MockAssociation) Clear() *ExecWrapper {
	a.operation = Replace
	a.noopAssociation.Clear()
	expectation := &SqlmockExecExpectation{parent: a.parent}

	return &ExecWrapper{association: a, expectation: expectation}
}

// Replace wraps gorm.Association.Replace
func (a *MockAssociation) Replace(values ...interface{}) *ExecWrapper {
	a.operation = Replace
	a.parent.noop.ReturnExecResult(1, 1)
	a.noopAssociation.Replace(values...)
	expectation := &SqlmockExecExpectation{parent: a.parent}

	return &ExecWrapper{association: a, expectation: expectation}
}

// Count wraps gorm.Association.Count
func (a *MockAssociation) Count() *QueryWrapper {
	a.noopAssociation.Count()
	expectation := &SqlmockQueryExpectation{parent: a.parent}

	return &QueryWrapper{association: a, expectation: expectation}
}

func handleAssociationGeneric(association *MockAssociation) {
	expecter := association.parent
	adapter := association.parent.adapter
	value := association.parent.gorm.Value
	stmts := association.parent.recorder.stmts

	for i, stmt := range stmts {
		switch i {
		case 0:
			adapter.ExpectExec(stmt).WillSucceed(1, 1)
		case 1:
			newExpecter := expecter.clone()
			newExpecter.recorder.stmts = []Stmt{stmt}
			newExpecter.Find(&value).Returns(value)
		}
	}
}

func handleReplace(association *MockAssociation, isSuccessful bool) {
	expecter := association.parent
	stmts := association.parent.recorder.stmts
	adapter := association.parent.adapter
	value := association.parent.gorm.Value

	for i, stmt := range stmts {
		switch i {
		// INSERT
		case 0:
			adapter.ExpectExec(stmt).WillSucceed(1, 1)
		// SELECT
		case 1:
			newExpecter := expecter.clone()
			newExpecter.recorder.stmts = []Stmt{stmt}
			newExpecter.Find(&value).Returns(value)
		// UPDATE
		case 2:
			adapter.ExpectExec(stmt).WillSucceed(1, 1)
		default:
			panic(fmt.Sprintf("Replace should not generate more than three SQL statements, got %d", len(stmts)))
		}
	}
}
