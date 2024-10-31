package apm

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupTestDB() (*gorm.DB, error) {
	dsn := "root:root@tcp(127.0.0.1:3306)/goapm?charset=utf8mb4&parseTime=True&loc=Local"
	return NewGorm("test", dsn)
}

func Test_GORM_SELECT(t *testing.T) {
	db, err := setupTestDB()
	assert.Nil(t, err)

	t.Run("select without context should work", func(t *testing.T) {
		var user User
		result := db.First(&user, "uid = ?", "nonexistent")
		assert.Error(t, result.Error)
		assert.Equal(t, gorm.ErrRecordNotFound, result.Error)
	})

	t.Run("select with context should work", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		var user User
		result := db.WithContext(ctx).First(&user, "uid = ?", "nonexistent")
		assert.Error(t, result.Error)
		assert.Equal(t, gorm.ErrRecordNotFound, result.Error)
	})
}

func Test_GORM_INSERT(t *testing.T) {
	db, err := setupTestDB()
	assert.Nil(t, err)

	t.Run("insert without context should work", func(t *testing.T) {
		user := User{
			Uid:     uuid.NewString(),
			Name:    "John",
			Age:     18,
			Gender:  "male",
			Address: "Beijing",
			Phone:   "1234567890",
			Email:   "john@example.com",
			Salary:  10000,
		}
		result := db.Create(&user)
		assert.Nil(t, result.Error)

		var insertedUser User
		result = db.Where("uid = ?", user.Uid).First(&insertedUser)
		assert.Nil(t, result.Error)
		assert.Equal(t, user.Uid, insertedUser.Uid)
		assert.Equal(t, user.Name, insertedUser.Name)
	})

	t.Run("insert with context should work", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		user := User{
			Uid:     uuid.NewString(),
			Name:    "John",
			Age:     18,
			Gender:  "male",
			Address: "Beijing",
			Phone:   "1234567890",
			Email:   "john@example.com",
			Salary:  10000,
		}
		result := db.WithContext(ctx).Create(&user)
		assert.Nil(t, result.Error)

		var insertedUser User
		result = db.WithContext(ctx).Where("uid = ?", user.Uid).First(&insertedUser)
		assert.Nil(t, result.Error)
		assert.Equal(t, user.Uid, insertedUser.Uid)
		assert.Equal(t, user.Name, insertedUser.Name)
		assert.Equal(t, user.Age, insertedUser.Age)
		assert.Equal(t, user.Gender, insertedUser.Gender)
		assert.Equal(t, user.Address, insertedUser.Address)
		assert.Equal(t, user.Phone, insertedUser.Phone)
		assert.Equal(t, user.Email, insertedUser.Email)
		assert.Equal(t, user.Salary, insertedUser.Salary)
	})
}

func Test_GORM_UPDATE(t *testing.T) {
	db, err := setupTestDB()
	assert.Nil(t, err)

	const nameBefore = "John"
	const nameAfter = "Alice"
	t.Run("update without context should work", func(t *testing.T) {
		user := User{
			Uid:     uuid.NewString(),
			Name:    nameBefore,
			Age:     18,
			Gender:  "male",
			Address: "Beijing",
			Phone:   "1234567890",
			Email:   "john@example.com",
			Salary:  10000,
		}
		db.Create(&user)

		user.Name = nameAfter
		result := db.Where("uid = ?", user.Uid).Save(&user)
		assert.Nil(t, result.Error)

		var updatedUser User
		result = db.Where("uid = ?", user.Uid).First(&updatedUser)
		assert.Nil(t, result.Error)
		assert.Equal(t, nameAfter, updatedUser.Name)
	})

	t.Run("update with context should work", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		user := User{
			Uid:     uuid.NewString(),
			Name:    nameBefore,
			Age:     18,
			Gender:  "male",
			Address: "Beijing",
			Phone:   "1234567890",
			Email:   "john@example.com",
			Salary:  10000,
		}
		db.Create(&user)

		user.Name = nameAfter
		result := db.WithContext(ctx).Where("uid = ?", user.Uid).Save(&user)
		assert.Nil(t, result.Error)

		var updatedUser User
		result = db.WithContext(ctx).Where("uid = ?", user.Uid).First(&updatedUser)
		assert.Nil(t, result.Error)
		assert.Equal(t, nameAfter, updatedUser.Name)
	})
}

func Test_GORM_DELETE(t *testing.T) {
	db, err := setupTestDB()
	assert.Nil(t, err)

	t.Run("delete without context should work", func(t *testing.T) {
		user := User{
			Uid:     uuid.NewString(),
			Name:    "John",
			Age:     18,
			Gender:  "male",
			Address: "Beijing",
			Phone:   "1234567890",
			Email:   "john@example.com",
			Salary:  10000,
		}
		db.Create(&user)

		result := db.Delete(&User{}, "uid = ?", user.Uid)
		assert.Nil(t, result.Error)

		var deletedUser User
		result = db.Where("uid = ?", user.Uid).First(&deletedUser)
		assert.Error(t, result.Error)
		assert.Equal(t, gorm.ErrRecordNotFound, result.Error)
	})

	t.Run("delete with context should work", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		user := User{
			Uid:     uuid.NewString(),
			Name:    "John",
			Age:     18,
			Gender:  "male",
			Address: "Beijing",
			Phone:   "1234567890",
			Email:   "john@example.com",
			Salary:  10000,
		}
		db.Create(&user)

		result := db.WithContext(ctx).Delete(&User{}, "uid = ?", user.Uid)
		assert.Nil(t, result.Error)

		var deletedUser User
		result = db.WithContext(ctx).Where("uid = ?", user.Uid).First(&deletedUser)
		assert.Error(t, result.Error)
		assert.Equal(t, gorm.ErrRecordNotFound, result.Error)
	})
}
