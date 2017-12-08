package gormexpect

import (
	"reflect"

	"github.com/davecgh/go-spew/spew"
	"github.com/jinzhu/gorm"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

// AdapterFactory is a generic interface for arbitrary adapters that satisfy
// the interface. variadic args are passed to gorm.Open.
type AdapterFactory func(dialect string, args ...interface{}) (*gorm.DB, Adapter, error)

// Expecter is the exported struct used for setting expectations
type Expecter struct {
	// globally scoped expecter
	adapter  Adapter
	callmap  map[string][]interface{} // these get called after we get a value from `Returns`
	gorm     *gorm.DB
	noop     NoopController
	recorder *Recorder
}

// NewDefaultExpecter returns a Expecter powered by go-sqlmock
func NewDefaultExpecter() (*gorm.DB, *Expecter, error) {
	gormMock, adapter, err := NewSqlmockAdapter("sqlmock", "mock_gorm_dsn")

	if err != nil {
		return nil, nil, err
	}

	recorder := &Recorder{}
	noop, noopc, _ := NewNoopDB()
	gormNoop, _ := gorm.Open("sqlmock", noop)
	gormNoop = gormNoop.Set("gorm:recorder", recorder)

	gormNoop.Callback().Create().After("gorm:create").Register("gorm_expect:record_exec", recordExecCallback)
	gormNoop.Callback().Query().Before("gorm:preload").Register("gorm_expect:record_preload", recordPreloadCallback)
	gormNoop.Callback().Query().After("gorm:query").Register("gorm_expect:record_query", recordQueryCallback)
	gormNoop.Callback().RowQuery().After("gorm:row_query").Register("gorm_expect:record_row_query", recordQueryCallback)
	gormNoop.Callback().Update().After("gorm:update").Register("gorm_expect:record_update", recordExecCallback)
	gormNoop.Callback().Delete().After("gorm:delete").Register("gorm_expect:record_delete", recordExecCallback)

	return gormMock, &Expecter{
		adapter:  adapter,
		callmap:  make(map[string][]interface{}),
		gorm:     gormNoop,
		noop:     noopc,
		recorder: recorder,
	}, nil
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

// Debug logs out queries
func (h *Expecter) Debug() *Expecter {
	h.gorm = h.gorm.Debug()

	return h
}

// AssertExpectations checks if all expected Querys and Execs were satisfied.
func (h *Expecter) AssertExpectations() error {
	return h.adapter.AssertExpectations()
}

// Association starts association mode
func (h *Expecter) Association(column string) *MockAssociation {
	gormAssociation := h.gorm.Association(column)
	return NewMockAssociation(column, gormAssociation, h)
}

// Model sets scope.Value
func (h *Expecter) Model(model interface{}) *Expecter {
	h.gorm = h.gorm.Model(model)
	return h
}

// Begin starts a mock transaction
func (h *Expecter) Begin() TxBeginner {
	return h.adapter.ExpectBegin()
}

// Commit commits a mock transaction
func (h *Expecter) Commit() TxCommitter {
	return h.adapter.ExpectCommit()
}

// Rollback rollsback a mock transaction
func (h *Expecter) Rollback() TxRollback {
	return h.adapter.ExpectRollback()
}

/* CREATE */

// Create mocks insertion of a model into the DB
func (h *Expecter) Create(model interface{}) ExecExpectation {
	h.gorm = h.gorm.Create(model)
	return h.exec()
}

/* READ */

// Limit sets limit parameter on query
func (h *Expecter) Limit(limit int) *Expecter {
	clone := h.clone()
	clone.gorm = clone.gorm.Limit(limit)
	return clone
}

// Offset sets offset parameter on query
func (h *Expecter) Offset(offset int) *Expecter {
	clone := h.clone()
	clone.gorm = clone.gorm.Offset(offset)
	return clone
}

// Assign will merge the given struct into the scope's value
func (h *Expecter) Assign(attrs ...interface{}) *Expecter {
	clone := h.clone()
	clone.gorm = clone.gorm.Assign(attrs...)
	return clone
}

// Where sets a WHERE condition(s)
func (h *Expecter) Where(query interface{}, args ...interface{}) *Expecter {
	clone := h.clone()
	clone.gorm = clone.gorm.Where(query, args...)
	return clone
}

// Not sets a NOT condition(s)
func (h *Expecter) Not(query interface{}, args ...interface{}) *Expecter {
	clone := h.clone()
	clone.gorm = clone.gorm.Not(query, args...)

	return clone
}

// Preload clones the expecter and sets a preload condition on gorm.DB
func (h *Expecter) Preload(column string, conditions ...interface{}) *Expecter {
	h.gorm = h.gorm.Preload(column, conditions...)

	return h
}

// First triggers a Query
func (h *Expecter) First(out interface{}, where ...interface{}) QueryExpectation {
	var args []interface{}
	args = append(args, out)
	args = append(args, where...)
	h.callmap["First"] = args

	return h.query()
}

// FirstOrCreate slightly differs from the equivalent Gorm method. It takes an
// extra argument (returns). If out and returns have the same type, returns is
// copied into out and nil is returned. The INSERT is not executed.
// If returns is nil, a not found error is set, and an ExecExpectation is
// returned.
func (h *Expecter) FirstOrCreate(out interface{}, returns interface{}, where ...interface{}) ExecExpectation {
	// not found
	if returns == nil {
		h.noop.ReturnNilRows()
		h.gorm.FirstOrCreate(out)

		// return our empty row
		query, _ := h.recorder.GetFirst()
		h.adapter.ExpectQuery(query).Returns(sqlmock.NewRows([]string{}))

		return h.exec()
	}

	// check if out and returns are of the same type. The out and returns
	// types should never differ.
	outType := indirect(reflect.ValueOf(out)).Type()
	returnsType := indirect(reflect.ValueOf(returns)).Type()

	if outType != returnsType {
		panic(spew.Sprintf("out and returns should be of the same type. Got %s and %s.\r\n", outType.String(), returnsType.String()))
	}

	// execute FirstOrCreate
	h.gorm.FirstOrCreate(out)

	// respond appropriately to the first query
	query, _ := h.recorder.GetFirst()
	clone := h.new()
	clone.recorder.stmts = []Stmt{query}
	queryExpectation := SqlmockQueryExpectation{parent: clone, scope: (&gorm.Scope{}).New(returns)}
	h.adapter.ExpectQuery(query).Returns(queryExpectation.getDestRows(returns))

	// we need to deal with the UPDATE if there are more queries
	if !h.recorder.IsEmpty() {
		return h.exec()
	}
	// h.First(out, where...).Returns(out)
	return nil
}

// FirstOrInit is similar to FirstOrCreate
func (h *Expecter) FirstOrInit(out interface{}, returns interface{}, where ...interface{}) *Expecter {
	var args []interface{}
	args = append(args, out)
	args = append(args, where...)
	h.callmap["FirstOrInit"] = args

	h.query().Returns(returns)
	h.reset()

	return h
}

// Find triggers a Query
func (h *Expecter) Find(out interface{}, where ...interface{}) QueryExpectation {
	// store our call in the map
	var args []interface{}
	args = append(args, out)
	args = append(args, where...)
	h.callmap["Find"] = args

	return h.query()
}

// Count triggers a query
func (h *Expecter) Count(out interface{}) QueryExpectation {
	var args []interface{}
	args = append(args, out)
	h.callmap["Count"] = args

	return h.query()
}

/* UPDATE */

// Save mocks updating a record in the DB and will trigger db.Exec()
func (h *Expecter) Save(model interface{}) ExecExpectation {
	h.gorm.Save(model)
	return h.exec()
}

// Update mocks updating the given attributes in the DB
func (h *Expecter) Update(attrs ...interface{}) ExecExpectation {
	h.gorm.Update(attrs...)
	return h.exec()
}

// Updates does the same thing as Update, but with map or struct
func (h *Expecter) Updates(values interface{}, ignoreProtectedAttrs ...bool) ExecExpectation {
	h.gorm.Updates(values, ignoreProtectedAttrs...)
	return h.exec()
}

/* DELETE */

// Delete does the same thing as gorm.Delete
func (h *Expecter) Delete(model interface{}, where ...interface{}) ExecExpectation {
	h.gorm.Delete(model, where...)
	return h.exec()
}

// clone ensures that the original expecter does not have any unintended
// conditions set by Where/Not etc. Recorder is _not_ cloned.
func (h *Expecter) clone() *Expecter {
	return &Expecter{
		adapter:  h.adapter,
		callmap:  make(map[string][]interface{}),
		gorm:     h.gorm,
		recorder: h.recorder,
	}
}

// new resets the recorder instance as well.
func (h *Expecter) new() *Expecter {
	return &Expecter{
		adapter:  h.adapter,
		callmap:  make(map[string][]interface{}),
		gorm:     h.gorm,
		recorder: &Recorder{},
	}
}

func (h *Expecter) reset() {
	h.callmap = make(map[string][]interface{})
	h.recorder.stmts = []Stmt{}
}

// query returns a SqlmockQuery with the current DB state
// it is responsible for executing SQL against then noop DB
func (h *Expecter) query() QueryExpectation {
	return &SqlmockQueryExpectation{parent: h}
}

func (h *Expecter) exec() ExecExpectation {
	return &SqlmockExecExpectation{parent: h}
}
