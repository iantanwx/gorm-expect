package gormexpect_test

import (
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	expecter "github.com/iantanwx/gorm-expect"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
)

type User struct {
	Id           int64
	Age          int64
	Name         string `sql:"size:255"`
	Email        string
	Birthday     *time.Time // Time
	CreatedAt    time.Time  // CreatedAt: Time of record is created, will be insert automatically
	UpdatedAt    time.Time  // UpdatedAt: Time of record is updated, will be updated automatically
	Emails       []Email    // Embedded structs
	CreditCard   CreditCard
	Languages    []Language `gorm:"many2many:user_languages;"`
	PasswordHash []byte
}

type CreditCard struct {
	ID        int8
	Number    string
	UserId    sql.NullInt64
	CreatedAt time.Time `sql:"not null"`
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"column:deleted_time"`
}

type Email struct {
	Id        int16
	UserId    int
	Email     string `sql:"type:varchar(100);"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Language struct {
	gorm.Model
	Name  string
	Users []User `gorm:"many2many:user_languages;"`
}

type UserRepository struct {
	db *gorm.DB
}

func (r *UserRepository) Find(limit int, offset int) ([]User, error) {
	var users []User
	err := r.db.Limit(limit).Offset(offset).Find(&users).Error

	return users, err
}

func (r *UserRepository) FindByID(id int64) (User, error) {
	user := User{Id: id}
	err := r.db.Preload("Emails").Preload("CreditCard").Preload("Languages").Find(&user).Error
	return user, err
}

func TestNewDefaultExpecter(t *testing.T) {
	db, _, err := expecter.NewDefaultExpecter()
	//lint:ignore SA5001 just a mock
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}
}

func TestNewCustomExpecter(t *testing.T) {
	db, _, err := expecter.NewExpecter(expecter.NewSqlmockAdapter, "sqlmock", "mock_gorm_dsn")
	//lint:ignore SA5001 just a mock
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}
}

func TestQuery(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()

	if err != nil {
		t.Fatal(err)
	}

	expect.First(&User{})
	db.First(&User{})

	if err := expect.AssertExpectations(); err != nil {
		t.Error(err)
	}
}

func TestQueryReturn(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	in := User{Id: 1}
	out := User{Id: 1, Name: "jinzhu"}

	expect.First(&in).Returns(out)

	db.First(&in)

	if e := expect.AssertExpectations(); e != nil {
		t.Error(e)
	}

	if in.Name != "jinzhu" {
		t.Errorf("Expected %s, got %s", out.Name, in.Name)
	}

	if ne := reflect.DeepEqual(in, out); !ne {
		t.Errorf("Not equal")
	}
}

func TestQueryReturnInline(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	in := User{}
	out := User{Id: 1, Name: "some_guy"}

	expect.Find(&in, "name = ?", "some_guy").Returns(out)
	db.Find(&in, "name = ?", "some_guy")

	assert.Nil(t, expect.AssertExpectations())
	assert.Equal(t, in, out)
}

func TestFindStructDest(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	in := &User{Id: 1}

	expect.Find(in)
	db.Find(&User{Id: 1})

	if e := expect.AssertExpectations(); e != nil {
		t.Error(e)
	}
}

func TestFindSlice(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	in := []User{}
	out := []User{User{Id: 1, Name: "jinzhu"}, User{Id: 2, Name: "itwx"}}

	expect.Find(&in).Returns(&out)
	db.Find(&in)

	if e := expect.AssertExpectations(); e != nil {
		t.Error(e)
	}

	if ne := reflect.DeepEqual(in, out); !ne {
		t.Error("Expected equal slices")
	}
}

func TestMockPreloadHasMany(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	in := User{Id: 1}
	outEmails := []Email{Email{Id: 1, UserId: 1}, Email{Id: 2, UserId: 1}}
	out := User{Id: 1, Emails: outEmails}

	expect.Preload("Emails").Find(&in).Returns(out)
	db.Preload("Emails").Find(&in)

	if err := expect.AssertExpectations(); err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(in, out) {
		t.Error("In and out are not equal")
	}
}

func TestMockPreloadHasOne(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	in := User{Id: 1}
	out := User{Id: 1, CreditCard: CreditCard{Number: "12345678"}}

	expect.Preload("CreditCard").Find(&in).Returns(out)
	db.Preload("CreditCard").Find(&in)

	if err := expect.AssertExpectations(); err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(in, out) {
		t.Error("In and out are not equal")
	}
}

func TestMockPreloadMany2Many(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	in := User{Id: 1}
	languages := []Language{Language{Name: "ZH"}}
	out := User{Id: 1, Languages: languages}

	expect.Preload("Languages").Find(&in).Returns(out)
	db.Preload("Languages").Find(&in)

	if err := expect.AssertExpectations(); err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(in, out) {
		t.Error("In and out are not equal")
	}
}

func TestMockPreloadMultiple(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	creditCard := CreditCard{Number: "12345678"}
	languages := []Language{Language{Name: "ZH"}}

	in := User{Id: 1}
	out := User{Id: 1, Languages: languages, CreditCard: creditCard}

	expect.Preload("Languages").Preload("CreditCard").Find(&in).Returns(out)
	db.Preload("Languages").Preload("CreditCard").Find(&in)

	if err := expect.AssertExpectations(); err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(in, out) {
		t.Error("In and out are not equal")
	}
}

func TestMockCreateBasic(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	user := User{Name: "jinzhu"}
	expect.Create(&user).WillSucceed(1, 1)
	rowsAffected := db.Create(&user).RowsAffected

	if rowsAffected != 1 {
		t.Errorf("Expected rows affected to be 1 but got %d", rowsAffected)
	}

	if user.Id != 1 {
		t.Errorf("User id field should be 1, but got %d", user.Id)
	}
}

func TestMockCreateError(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	mockError := errors.New("Could not insert user")

	user := User{Name: "jinzhu"}
	expect.Create(&user).WillFail(mockError)

	dbError := db.Create(&user).Error

	if dbError == nil || dbError != mockError {
		t.Errorf("Expected *DB.Error to be set, but it was not")
	}
}

func TestMockSaveBasic(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	user := User{Name: "jinzhu"}
	expect.Save(&user).WillSucceed(1, 1)
	expected := db.Save(&user)

	if err := expect.AssertExpectations(); err != nil {
		t.Errorf("Expectations were not met %s", err.Error())
	}

	if expected.RowsAffected != 1 || user.Id != 1 {
		t.Errorf("Expected result was not returned")
	}
}

// func TestMockUpdateBasic(t *testing.T) {
// 	db, expect, err := expecter.NewDefaultExpecter()
// 	defer db.Close()

// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	newName := "uhznij"
// 	user := User{Name: "jinzhu"}

// 	expect.Model(&user).Update("name", newName).WillSucceed(1, 1)
// 	db.Model(&user).Update("name", newName)

// 	if err := expect.AssertExpectations(); err != nil {
// 		t.Errorf("Expectations were not met %s", err.Error())
// 	}

// 	if user.Name != newName {
// 		t.Errorf("Should have name %s but got %s", newName, user.Name)
// 	}
// }

// func TestMockUpdatesBasic(t *testing.T) {
// 	db, expect, err := expecter.NewDefaultExpecter()
// 	defer db.Close()

// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	user := User{Name: "jinzhu", Age: 18}
// 	updated := User{Name: "jinzhu", Age: 88}

// 	expect.Model(&user).Updates(updated).WillSucceed(1, 1)
// 	db.Model(&user).Updates(updated)

// 	if err := expect.AssertExpectations(); err != nil {
// 		t.Errorf("Expectations were not met %s", err.Error())
// 	}

// 	if user.Age != updated.Age {
// 		t.Errorf("Should have age %d but got %d", user.Age, updated.Age)
// 	}
// }

func TestUserRepoFind(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	repo := &UserRepository{db}

	expected := []User{User{Name: "my_name"}}

	expect.Find(&[]User{}).Returns(expected)
	users, err := repo.Find(1, 0)

	assert.Nil(t, err)
	assert.Nil(t, expect.AssertExpectations())
	assert.Equal(t, expected, users)
}

func TestUserRepoPreload(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	repo := &UserRepository{db}

	// has one
	creditCard := CreditCard{Number: "12345678"}
	// has many
	email := []Email{
		Email{Email: "fake_user@live.com"},
		Email{Email: "fake_user@gmail.com"},
	}
	// many to many
	languages := []Language{
		Language{Name: "EN"},
		Language{Name: "ZH"},
	}

	expected := User{
		Id:         1,
		Name:       "my_name",
		CreditCard: creditCard,
		Emails:     email,
		Languages:  languages,
	}

	expect.Preload("Emails").Preload("CreditCard").Preload("Languages").Find(&User{Id: 1}).Returns(expected)
	actual, err := repo.FindByID(1)

	assert.Nil(t, expect.AssertExpectations())
	assert.Nil(t, err)
	assert.Equal(t, expected, actual)
}
