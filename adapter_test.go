package gormexpect_test

import (
	"errors"
	"testing"

	expecter "github.com/iantanwx/gorm-expect"
	"github.com/stretchr/testify/assert"
)

func TestWillFail(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	expected := errors.New("failed")
	expect.Create(&User{}).WillFail(expected)

	assert.Equal(t, expected, db.Create(&User{}).Error)
}
