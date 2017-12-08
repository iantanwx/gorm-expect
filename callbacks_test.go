package gormexpect_test

import (
	"testing"

	expecter "github.com/iantanwx/gorm-expect"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
)

var isCalled int

type TestCallbackModel struct {
	gorm.Model
}

func (m *TestCallbackModel) BeforeDelete() {
	isCalled++
}

func TestSkipCallbacks(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	model := TestCallbackModel{}
	model.ID = 1

	expect.Skip().Delete(&model).WillSucceed(1, 1)
	db.Delete(&model)

	assert.Equal(t, 1, isCalled)
}
