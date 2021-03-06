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
	Company      string `sql:"default:'Tech in Asia'"`
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
	ID        int
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

func (r *UserRepository) FindUser(statement string, vars ...interface{}) (User, error) {
	var user User
	err := r.db.Preload("Emails").Preload("CreditCard").Where(statement, vars...).Find(&user).Error
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

func TestInlineQuery(t *testing.T) {
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

func TestStructDest(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	actual := User{Id: 1}
	expected := User{Id: 1, Name: "jinzhu"}

	expect.Find(&actual).Returns(expected)
	db.Find(&actual)

	assert.Nil(t, expect.AssertExpectations())
	assert.Equal(t, expected, actual)
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

	assert.Nil(t, expect.AssertExpectations())
	assert.Equal(t, out, in)
}

func TestOffsetLimit(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	users := []User{}

	expect.Limit(10).Offset(1).Find(&users).Returns(nil)
	db.Limit(10).Offset(1).Find(&users)

	assert.Nil(t, expect.AssertExpectations())
}

func TestNot(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	user1 := User{}
	user2 := User{}

	expect.Not("name", "jinzhu").Find(&user1).Returns(User{Id: 1, Name: "not_jinzhu"})
	db.Not("name", "jinzhu").Find(&user1)

	// no pollution of conditions
	expect.Find(&user2).Returns(User{Id: 2, Name: "jinzhu"})
	err = db.Find(&user2).Error

	assert.Nil(t, expect.AssertExpectations())
	assert.Nil(t, err)
}

func TestCount(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	var actual int64
	var expected int64 = 5

	expect.Model(User{}).Where("name = ?", "jinzhu").Count(&actual).Returns(5)
	db.Model(User{}).Where("name = ?", "jinzhu").Count(&actual)

	assert.Nil(t, expect.AssertExpectations())
	assert.Equal(t, expected, actual)
}

func TestPreloadHasMany(t *testing.T) {
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

	assert.Nil(t, expect.AssertExpectations())
	assert.Equal(t, out, in)
}

func TestPreloadHasOne(t *testing.T) {
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

func TestPreloadMany2Many(t *testing.T) {
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

func TestPreloadMany2ManyEmpty(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	languages := []Language{}
	user := User{Id: 1, Languages: languages}

	expect.Preload("Languages").Find(&user).Returns(user)
	err = db.Preload("Languages").Find(&user).Error

	if err := expect.AssertExpectations(); err != nil {
		t.Error(err)
	}

	assert.Nil(t, err)
}

func TestPreloadMultiple(t *testing.T) {
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

	assert.Nil(t, expect.AssertExpectations())
	assert.Equal(t, out, in)
}

func TestCreate(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	user := User{Name: "jinzhu"}
	expect.Create(&user).WillSucceed(1, 1)
	db = db.Create(&user)

	var expected int64 = 1

	assert.Nil(t, expect.AssertExpectations())
	assert.Nil(t, db.Error)
	assert.Equal(t, expected, db.RowsAffected)
	assert.Equal(t, expected, user.Id)
}

func TestCreateMany(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	names := []string{"jinzhu", "uhzni", "jack", "jill"}

	for _, name := range names {
		expect.Create(&User{Name: name}).WillSucceed(1, 1)
		assert.Nil(t, db.Create(&User{Name: name}).Error)
	}

	assert.Nil(t, expect.AssertExpectations())
}

func TestCreateError(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	mockError := errors.New("Could not insert user")

	user := User{Name: "jinzhu"}
	expect.Create(&user).WillFail(mockError)

	dbError := db.Create(&user).Error

	assert.Error(t, dbError)
	assert.Equal(t, mockError, dbError)
}

func TestSaveBasic(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	user := User{Name: "jinzhu"}
	expect.Save(&user).WillSucceed(1, 1)
	expected := db.Save(&user)
	var expectedRows int64 = 1
	var expectedID int64 = 1

	assert.Nil(t, expect.AssertExpectations())
	assert.Equal(t, expectedRows, expected.RowsAffected)
	assert.Equal(t, expectedID, user.Id)
}

func TestUpdate(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	newName := "uhznij"
	user := User{Name: "jinzhu"}

	expect.Model(&user).Update("name", newName).WillSucceed(1, 1)
	err = db.Model(&user).Update("name", newName).Error

	assert.Nil(t, expect.AssertExpectations())
	assert.NoError(t, err)
	assert.Equal(t, newName, user.Name)
}

func TestUpdatesBasic(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	user := User{Name: "jinzhu", Age: 18}
	updated := User{Name: "jinzhu", Age: 88}

	expect.Model(&user).Updates(updated).WillSucceed(1, 1)
	db.Model(&user).Updates(updated)

	assert.Equal(t, user.Age, updated.Age)
	assert.Nil(t, expect.AssertExpectations())
}

func TestFirstOrCreateExisting(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	user := User{Id: 1, Name: "jinzhu", Age: 18}

	expect.Where("id = ?", 1).FirstOrCreate(&user, user)
	err = db.Where("id = ?", 1).FirstOrCreate(&user).Error

	assert.Nil(t, expect.AssertExpectations())
	assert.Nil(t, err)
}

func TestFirstOrCreateSuccess(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	user := User{Id: 1, Name: "jinzhu", Age: 18}

	expect.FirstOrCreate(&user, nil).WillSucceed(1, 1)
	db.FirstOrCreate(&user)

	assert.Nil(t, expect.AssertExpectations())
}

func TestFirstOrCreateAssign(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	user := User{Id: 1, Name: "jinzhu", Age: 18}
	update := User{Email: "somebody@gmail.com"}

	expect.Assign(update).FirstOrCreate(&user, user).WillSucceed(1, 1)

	assert.NoError(t, db.Assign(update).FirstOrCreate(&user).Error)
	assert.Nil(t, expect.AssertExpectations())
}

func TestFirstOrCreateWrongTypes(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	user := User{Id: 1, Name: "jinzhu", Age: 18}

	assert.Panics(t, func() { expect.FirstOrCreate(&user, Email{}).WillSucceed(1, 1) })
}

func TestDeleteSuccess(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	user := &User{Id: 1, Name: "jinzhu"}

	expect.Delete(&user, "id = ?", 1).WillSucceed(1, 1)
	result := db.Delete(&user, "id = ?", 1)

	assert.Nil(t, expect.AssertExpectations())
	assert.Nil(t, result.Error)
	assert.Equal(t, int64(1), result.RowsAffected)
}

func TestDeleteError(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	user := &User{Id: 1, Name: "jinzhu"}

	expect.Delete(&user, "id = ?", 1).WillFail(errors.New("Could not delete"))
	rowsAffected := db.Delete(&user, "id = ?", 1).RowsAffected

	assert.Nil(t, expect.AssertExpectations())
	assert.Equal(t, int64(0), rowsAffected)
}

func TestSoftDelete(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	// use Language because User doesn't have a soft delete field
	language := Language{Name: "EN"}
	language.ID = 1

	expect.Delete(&language).WillSucceed(1, 1)
	db.Delete(&language)

	assert.Nil(t, expect.AssertExpectations())
}

func TestFirstOrInitNil(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	in1 := User{}
	in2 := User{}
	expected := User{Id: 1, Name: "jinzhu", Age: 18}

	expect.FirstOrInit(&in1, nil, expected)
	db.FirstOrInit(&in2, expected)

	assert.Nil(t, expect.AssertExpectations())
	assert.Equal(t, in1.Id, in2.Id)
	assert.Equal(t, in1.Name, in2.Name)
	assert.Equal(t, in1.Age, in2.Age)
}

func TestFirstOrInitChain(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	in1 := User{}
	in2 := User{}

	expected := User{Id: 1, Name: "jinzhu", Age: 18}
	var expectedRowsAffected int64 = 1

	expect.FirstOrInit(&in1, nil, expected).Create(&in1).WillSucceed(1, 1)
	rowsAffected := db.FirstOrInit(&in2, expected).Create(&in2).RowsAffected

	assert.Nil(t, expect.AssertExpectations())
	assert.Equal(t, in1.Id, in2.Id)
	assert.Equal(t, in1.Name, in2.Name)
	assert.Equal(t, in1.Age, in2.Age)
	assert.Equal(t, expectedRowsAffected, rowsAffected)
}

func TestFirstOrInitFound(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	in1 := User{}
	in2 := User{}

	expected := User{Id: 1, Name: "jinzhu", Age: 18}

	expect.FirstOrInit(&in1, expected, "id = ?", 1)
	db.FirstOrInit(&in2, "id = ?", 1)

	assert.Nil(t, expect.AssertExpectations())
	assert.Equal(t, expected, in2)
}

func TestUserRepoFind(t *testing.T) {
	db, expect, err := expecter.NewDefaultExpecter()
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}

	repo := &UserRepository{db}

	expected := []User{User{Name: "my_name"}}

	expect.Limit(1).Offset(0).Find(&[]User{}).Returns(expected)
	users, err := repo.Find(1, 0)

	assert.Nil(t, err)
	assert.Nil(t, expect.AssertExpectations())
	assert.Equal(t, expected, users)
}

func TestUserRepoPreload1(t *testing.T) {
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

func TestUserRepoPreload2(t *testing.T) {
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

	expected := User{
		Id:         1,
		Name:       "my_name",
		CreditCard: creditCard,
		Emails:     email,
	}

	expect.Preload("Emails").Preload("CreditCard").Where("id = ?", 1).Find(&User{}).Returns(expected)
	actual, err := repo.FindUser("id = ?", 1)

	assert.Nil(t, expect.AssertExpectations())
	assert.Nil(t, err)
	assert.Equal(t, expected, actual)
}
