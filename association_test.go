package gormexpect_test

import (
	"testing"

	expecter "github.com/iantanwx/gorm-expect"
	"github.com/icrowley/fake"
	"github.com/stretchr/testify/assert"
)

func TestAssociationModeFind(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	var emails []Email
	user := &User{Id: 1, Name: "jinzhu"}

	expect.Model(&user).Association("Emails").Find(&emails).Returns([]Email{Email{Email: "jinzhu@gmail.com"}})
	err = db.Model(&user).Association("Emails").Find(&emails).Error

	assert.Nil(t, expect.AssertExpectations())
}

func TestAssociationModeAppend(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	user1 := User{Id: 1, Name: "jinzhu"}
	user2 := User{Id: 1, Name: "jinzhu"}
	emails := []Email{Email{UserId: 1, Email: "uhznij@liamg.moc"}}

	expect.Model(&user1).Association("Emails").Append(emails).WillSucceed(1, 1)
	err = db.Model(&user2).Association("Emails").Append(emails).Error

	assert.Nil(t, expect.AssertExpectations())
	assert.Nil(t, err)
}

func TestAssociationModeDelete(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	emails := []Email{Email{UserId: 1, Email: "uhznij@liamg.moc"}}
	user1 := User{Id: 1, Name: "jinzhu"}
	user2 := User{Id: 1, Name: "jinzhu"}

	expect.Model(&user1).Association("Emails").Delete(emails).WillSucceed(1, 1)
	err = db.Model(&user2).Association("Emails").Delete(emails).Error

	assert.Nil(t, expect.AssertExpectations())
	assert.Nil(t, err)
	assert.Nil(t, user2.Emails)
}

func TestAssociationModeDeleteM2M(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	languages := []Language{}
	for i := 0; i < 10; i++ {
		language := Language{Name: fake.Language()}
		language.ID = uint(i) + uint(1)
		languages = append(languages, language)
	}

	expect.Model(&User{Id: 1}).Association("Languages").Delete(languages).WillSucceed(10, 10)
	err = db.Model(&User{Id: 1}).Association("Languages").Delete(languages).Error

	t.Log(expect.AssertExpectations())
}

func TestAssociationModeClear(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	emails := []Email{Email{UserId: 1, Email: "uhznij@liamg.moc"}}
	user := User{Id: 1, Name: "jinzhu", Emails: emails}

	expect.Model(&user).Association("Emails").Clear().WillSucceed(1, 1)
	err = db.Model(&user).Association("Emails").Clear().Error

	assert.Nil(t, expect.AssertExpectations())
	assert.Nil(t, err)
	assert.Equal(t, 0, len(user.Emails))
}

func TestAssociationModeReplace(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	oldEmails := []Email{
		Email{UserId: 1, Email: "jinzhu@gmail.com"},
		Email{UserId: 1, Email: "uhznij@liamg.moc"},
	}
	newEmails := []Email{
		Email{UserId: 1, Email: "jinzhu@gmail.com"},
	}
	user := User{Id: 1, Name: "jinzhu", Emails: oldEmails}

	expect.Model(&user).Association("Emails").Replace(newEmails).WillSucceed(1, 1)
	err = db.Model(&user).Association("Emails").Replace(newEmails).Error

	assert.Nil(t, expect.AssertExpectations())
	assert.Nil(t, err)
	assert.Equal(t, 1, len(user.Emails))
}

func TestAssociationModeCount(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	user := User{Id: 1}

	expect.Model(&user).Association("Emails").Count().Returns(5)
	count := db.Model(&user).Association("Emails").Count()

	assert.Nil(t, expect.AssertExpectations())
	assert.Equal(t, 5, count)
}
