package gormexpect_test

import (
	"testing"

	expecter "github.com/iantanwx/gorm-expect"
	"github.com/stretchr/testify/assert"
)

func TestTransactionBegin(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	expect.Begin()
	db.Begin()

	assert.Nil(t, expect.AssertExpectations())
}

func TestTransactionCommit(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	expect.Begin()
	expect.Commit()
	db.Begin().Commit()

	assert.Nil(t, expect.AssertExpectations())
}

func TestTransactionRollback(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	expect.Begin()
	expect.Rollback()

	db.Begin().Rollback()

	assert.Nil(t, expect.AssertExpectations())
}
