package gormexpect

import (
	"github.com/jinzhu/gorm"
)

// MockAssociation mirros gorm.Association
type MockAssociation struct {
	column          string
	parent          *Expecter
	noopAssociation *gorm.Association
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
func (w *ExecWrapper) WillSucceed(lastReturnID, rowsAffected int64) *QueryWrapper {
	// execute INSERT first
	w.expectation.WillSucceed(lastReturnID, rowsAffected)

	// force the second query becuase we don't record it
	value := w.association.parent.gorm.Value
	expectation := w.association.parent.Find(&value)

	return &QueryWrapper{association: w.association, expectation: expectation}
}

// NewMockAssociation returns a MockAssociation
func NewMockAssociation(c string, a *gorm.Association, e *Expecter) *MockAssociation {
	return &MockAssociation{c, e, a}
}

// Find wraps gorm.Association
func (a *MockAssociation) Find(value interface{}) *QueryWrapper {
	a.noopAssociation.Find(&value)
	expectation := &SqlmockQueryExpectation{association: a, parent: a.parent}

	return &QueryWrapper{association: a, expectation: expectation}
}

// Append wraps gorm.Association.Append
func (a *MockAssociation) Append(values ...interface{}) *ExecWrapper {
	a.noopAssociation.Append(values...)
	expectation := &SqlmockExecExpectation{parent: a.parent}

	return &ExecWrapper{association: a, expectation: expectation}
}

// Delete wraps gorm.Association.Delete
func (a *MockAssociation) Delete(values ...interface{}) *ExecWrapper {
	a.noopAssociation.Delete(values...)
	expectation := &SqlmockExecExpectation{parent: a.parent}

	return &ExecWrapper{association: a, expectation: expectation}
}
