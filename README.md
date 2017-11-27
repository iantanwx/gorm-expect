## Why

Testing `gorm`-based DALs is terrible.

## Installation

```
go get -u github.com/iantanwx/gorm-expect
```

## Usage

The API reflects Gorm's API (mostly).

```
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
```
