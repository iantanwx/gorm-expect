package gormexpect

import (
	"github.com/jinzhu/gorm"
)

// AdapterFactory is a generic interface for arbitrary adapters that satisfy
// the interface. variadic args are passed to gorm.Open.
type AdapterFactory func(dialect string, args ...interface{}) (*gorm.DB, Adapter, error)

// Expecter is the exported struct used for setting expectations
type Expecter struct {
	// globally scoped expecter
	adapter  Adapter
	gorm     *gorm.DB
	recorder *Recorder
}

// NewDefaultExpecter returns a Expecter powered by go-sqlmock
func NewDefaultExpecter() (*gorm.DB, *Expecter, error) {
	gormMock, adapter, err := NewSqlmockAdapter("sqlmock", "mock_gorm_dsn")

	if err != nil {
		return nil, nil, err
	}

	recorder := &Recorder{}
	noop, _ := NewNoopDB()
	gormNoop, _ := gorm.Open("sqlmock", noop)
	gormNoop = gormNoop.Set("gorm:recorder", recorder)

	gormNoop.Callback().Create().After("gorm:create").Register("gorm_expect:record_exec", recordExecCallback)
	gormNoop.Callback().Query().Before("gorm:preload").Register("gorm_expect:record_preload", recordPreloadCallback)
	gormNoop.Callback().Query().After("gorm:query").Register("gorm_expect:record_query", recordQueryCallback)
	gormNoop.Callback().RowQuery().After("gorm:row_query").Register("gorm_expect:record_query", recordQueryCallback)
	gormNoop.Callback().Update().After("gorm:update").Register("gorm_expect:record_exec", recordExecCallback)

	return gormMock, &Expecter{adapter: adapter, gorm: gormNoop, recorder: recorder}, nil
}

// NewExpecter returns an Expecter for arbitrary adapters
func NewExpecter(fn AdapterFactory, dialect string, args ...interface{}) (*gorm.DB, *Expecter, error) {
	gormDb, adapter, err := fn(dialect, args...)

	if err != nil {
		return nil, nil, err
	}

	return gormDb, &Expecter{adapter: adapter}, nil
}

/* PUBLIC METHODS */

// AssertExpectations checks if all expected Querys and Execs were satisfied.
func (h *Expecter) AssertExpectations() error {
	return h.adapter.AssertExpectations()
}

// Model sets scope.Value
func (h *Expecter) Model(model interface{}) *Expecter {
	h.gorm = h.gorm.Model(model)
	return h
}

/* CREATE */

// Create mocks insertion of a model into the DB
func (h *Expecter) Create(model interface{}) ExpectedExec {
	h.gorm = h.gorm.Create(model)
	return h.adapter.ExpectExec(h.recorder.stmts[0])
}

/* READ */

// Where sets a condition
func (h *Expecter) Where(query interface{}, args ...interface{}) *Expecter {
	h.gorm = h.gorm.Where(query, args...)
	return h
}

// Preload clones the expecter and sets a preload condition on gorm.DB
func (h *Expecter) Preload(column string, conditions ...interface{}) *Expecter {
	h.gorm = h.gorm.Preload(column, conditions...)

	return h
}

// First triggers a Query
func (h *Expecter) First(out interface{}, where ...interface{}) ExpectedQuery {
	h.gorm = h.gorm.First(out, where...)
	return h.adapter.ExpectQuery(h.recorder.stmts...)
}

// Find triggers a Query
func (h *Expecter) Find(out interface{}, where ...interface{}) ExpectedQuery {
	h.gorm = h.gorm.Find(out, where...)
	return h.adapter.ExpectQuery(h.recorder.stmts...)
}

/* UPDATE */

// Save mocks updating a record in the DB and will trigger db.Exec()
func (h *Expecter) Save(model interface{}) ExpectedExec {
	h.gorm.Save(model)
	return h.adapter.ExpectExec(h.recorder.stmts[0])
}

// Update mocks updating the given attributes in the DB
func (h *Expecter) Update(attrs ...interface{}) ExpectedExec {
	h.gorm.Update(attrs...)
	return h.adapter.ExpectExec(h.recorder.stmts[0])
}

// Updates does the same thing as Update, but with map or struct
func (h *Expecter) Updates(values interface{}, ignoreProtectedAttrs ...bool) ExpectedExec {
	h.gorm.Updates(values, ignoreProtectedAttrs...)
	return h.adapter.ExpectExec(h.recorder.stmts[0])
}
