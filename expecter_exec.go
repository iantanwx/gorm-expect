package gormexpect

// ExecExpectation is returned by Expecter. It exposes a narrower API than
// Execer to limit footguns.
type ExecExpectation interface {
	WillSucceed(lastReturnedID, rowsAffected int64) ExecExpectation
	WillFail(err error) ExecExpectation
}

// SqlmockExecExpectation implement ExecExpectation with gosqlmock
type SqlmockExecExpectation struct {
	parent *Expecter
}

// WillSucceed sets the exec to be successful with the passed ID and rows
// affected
func (e *SqlmockExecExpectation) WillSucceed(lastReturnedID, rowsAffected int64) ExecExpectation {
	query, _ := e.parent.recorder.GetFirst()
	e.parent.adapter.ExpectExec(query).WillSucceed(lastReturnedID, rowsAffected)

	return e
}

// WillFail sets the exec to fail with the passed error
func (e *SqlmockExecExpectation) WillFail(err error) ExecExpectation {
	query, _ := e.parent.recorder.GetFirst()
	e.parent.adapter.ExpectExec(query).WillFail(err)

	return e
}
