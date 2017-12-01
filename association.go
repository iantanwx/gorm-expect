package gormexpect

import (
	"github.com/jinzhu/gorm"
)

// MockAssociation mirros gorm.Association
type MockAssociation struct {
	parent          *Expecter
	noopAssociation *gorm.Association
	callmap         map[string][]interface{}
}

// NewMockAssociation returns a MockAssociation
func NewMockAssociation(a *gorm.Association, e *Expecter) *MockAssociation {
	return &MockAssociation{e, a, make(map[string][]interface{})}
}

// Find wraps gorm.Association
func (a *MockAssociation) Find(value interface{}) QueryExpectation {
	a.noopAssociation.Find(&value)
	return &SqlmockQueryExpectation{parent: a.parent}
}
