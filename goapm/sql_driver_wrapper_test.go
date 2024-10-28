package goapm

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type User struct {
	Uid     string  `gorm:"column:uid;unique"`
	Name    string  `gorm:"column:name"`
	Age     int     `gorm:"column:age"`
	Gender  string  `gorm:"column:gender"`
	Address string  `gorm:"column:address"`
	Phone   string  `gorm:"column:phone"`
	Email   string  `gorm:"column:email"`
	Salary  float64 `gorm:"column:salary"`
}

func (u *User) TableName() string {
	return "t_user"
}

func Test_SQLDriverWrapper_SELECT(t *testing.T) {
	db, err := NewMySQL("root:root@tcp(127.0.0.1:3306)/goapm")
	assert.Nil(t, err)
	defer db.Close()
	t.Run("select without context should work", func(t *testing.T) {
		var result string
		err := db.QueryRow("SELECT 1").Scan(&result)
		assert.Nil(t, err)
		assert.Equal(t, "1", result)
	})

	t.Run("select with context should work", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		var result string
		err := db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
		assert.Nil(t, err)
		assert.Equal(t, "1", result)
	})

	t.Run("select with context but not exists should work", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		var result User
		uid := uuid.NewString()
		err := db.QueryRowContext(ctx, "SELECT `uid`, `name`, `age`, `gender`, `address`, `phone`, `email`, `salary` from `t_user` "+
			"where `uid` = ?", uid).Scan(
			&result.Uid,
			&result.Name,
			&result.Age,
			&result.Gender,
			&result.Address,
			&result.Phone,
			&result.Email,
			&result.Salary,
		)
		assert.Equal(t, sql.ErrNoRows, err)
		assert.Equal(t, "", result.Uid)
	})
}

func Test_SQLDriverWrapper_INSERT(t *testing.T) {
	db, err := NewMySQL("root:root@tcp(127.0.0.1:3306)/goapm")
	assert.Nil(t, err)
	defer db.Close()

	t.Run("insert without context should work", func(t *testing.T) {
		uid := uuid.NewString()
		_, err := db.Exec("INSERT INTO `t_user` (`uid`, `name`, `age`, `gender`, `address`, `phone`, `email`, `salary`)"+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?)", uid, "John", 18, "male", "Beijing", "1234567890", "john@example.com", 10000)
		assert.Nil(t, err)

		var result User
		err = db.QueryRow("SELECT `uid`, `name`, `age`, `gender`, `address`, `phone`, `email`, `salary` FROM `t_user` WHERE `uid` = ?", uid).Scan(
			&result.Uid,
			&result.Name,
			&result.Age,
			&result.Gender,
			&result.Address,
			&result.Phone,
			&result.Email,
			&result.Salary,
		)
		assert.Nil(t, err)
		assert.Equal(t, uid, result.Uid)
		assert.Equal(t, "John", result.Name)
		assert.Equal(t, 18, result.Age)
		assert.Equal(t, "male", result.Gender)
		assert.Equal(t, "Beijing", result.Address)
		assert.Equal(t, "1234567890", result.Phone)
		assert.Equal(t, "john@example.com", result.Email)
		assert.Equal(t, float64(10000), result.Salary)
	})

	t.Run("insert with context should work", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		uid := uuid.NewString()
		_, err := db.ExecContext(ctx, "INSERT INTO `t_user` (`uid`, `name`, `age`, `gender`, `address`, `phone`, `email`, `salary`)"+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?)", uid, "John", 18, "male", "Beijing", "1234567890", "john@example.com", 10000)
		assert.Nil(t, err)

		var result string
		err = db.QueryRowContext(ctx, "SELECT `uid` FROM `t_user` WHERE `uid` = ?", uid).Scan(&result)
		assert.Nil(t, err)
		assert.Equal(t, uid, result)
	})

	t.Run("insert with context and duplicated uid should work", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
		defer cancel()

		uid := "u001"
		_, err := db.ExecContext(ctx, "INSERT INTO `t_user` (`uid`, `name`, `age`, `gender`, `address`, `phone`, `email`, `salary`)"+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?)", uid, "John", 18, "male", "Beijing", "1234567890", "john@example.com", 10000)
		assert.True(t, strings.Contains(err.Error(), "Duplicate entry"))
	})
}

func Test_SQLDriverWrapper_UPDATE(t *testing.T) {
	db, err := NewMySQL("root:root@tcp(127.0.0.1:3306)/goapm")
	assert.Nil(t, err)
	defer db.Close()

	t.Run("update without context should work", func(t *testing.T) {
		const nameBefore = "John"
		const nameAfter = "Alice"

		uid := uuid.NewString()
		_, err := db.Exec("INSERT INTO `t_user` (`uid`, `name`, `age`, `gender`, `address`, `phone`, `email`, `salary`)"+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?)", uid, nameBefore, 18, "male", "Beijing", "1234567890", "john@example.com", 10000)
		assert.Nil(t, err)

		_, err = db.Exec("UPDATE `t_user` SET `name` = ? WHERE `uid` = ?", nameAfter, uid)
		assert.Nil(t, err)

		var result User
		err = db.QueryRow("SELECT  `name` FROM `t_user` WHERE `uid` = ?", uid).Scan(
			&result.Name,
		)
		assert.Nil(t, err)
		assert.Equal(t, nameAfter, result.Name)
	})

	t.Run("update with context should work", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
		defer cancel()

		const nameBefore = "John"
		const nameAfter = "Alice"

		uid := uuid.NewString()
		_, err := db.ExecContext(ctx, "INSERT INTO `t_user` (`uid`, `name`, `age`, `gender`, `address`, `phone`, `email`, `salary`)"+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?)", uid, nameBefore, 18, "male", "Beijing", "1234567890", "john@example.com", 10000)
		assert.Nil(t, err)

		_, err = db.ExecContext(ctx, "UPDATE `t_user` SET `name` = ? WHERE `uid` = ?", nameAfter, uid)
		assert.Nil(t, err)

		var result string
		err = db.QueryRowContext(ctx, "SELECT `name` FROM `t_user` WHERE `uid` = ?", uid).Scan(&result)
		assert.Nil(t, err)
		assert.Equal(t, nameAfter, result)
	})
}

func Test_SQLDriverWrapper_DELETE(t *testing.T) {
	db, err := NewMySQL("root:root@tcp(127.0.0.1:3306)/goapm")
	assert.Nil(t, err)
	defer db.Close()

	t.Run("delete without context and exists uid should work", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
		defer cancel()

		uid := uuid.NewString()
		_, err := db.Exec("INSERT INTO `t_user` (`uid`, `name`, `age`, `gender`, `address`, `phone`, `email`, `salary`)"+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?)", uid, "John", 18, "male", "Beijing", "1234567890", "john@example.com", 10000)
		assert.Nil(t, err)

		var result string
		err = db.QueryRowContext(ctx, "SELECT `uid` FROM `t_user` WHERE `uid` = ?", uid).Scan(&result)
		assert.Nil(t, err)
		assert.Equal(t, uid, result)

		_, err = db.Exec("DELETE FROM `t_user` WHERE `uid` = ?", uid)
		assert.Nil(t, err)

		result = ""
		err = db.QueryRowContext(ctx, "SELECT `uid` FROM `t_user` WHERE `uid` = ?", uid).Scan(&result)
		assert.Equal(t, sql.ErrNoRows, err)
		assert.Equal(t, "", result)
	})

	t.Run("delete with context should work", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
		defer cancel()

		uid := uuid.NewString()
		_, err := db.ExecContext(ctx, "INSERT INTO `t_user` (`uid`, `name`, `age`, `gender`, `address`, `phone`, `email`, `salary`)"+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?)", uid, "John", 18, "male", "Beijing", "1234567890", "john@example.com", 10000)
		assert.Nil(t, err)

		_, err = db.ExecContext(ctx, "DELETE FROM `t_user` WHERE `uid` = ?", uid)
		assert.Nil(t, err)

		var result string
		err = db.QueryRowContext(ctx, "SELECT `uid` FROM `t_user` WHERE `uid` = ?", uid).Scan(&result)
		assert.Equal(t, sql.ErrNoRows, err)
		assert.Equal(t, "", result)
	})

	t.Run("delete without context and not exists uid should work", func(t *testing.T) {
		uid := uuid.NewString()
		_, err := db.Exec("DELETE FROM `t_user` WHERE `uid` = ?", uid)
		assert.Nil(t, err)
	})
}

func Test_SQLDriverWrapper_Prepare(t *testing.T) {
	db, err := NewMySQL("root:root@tcp(127.0.0.1:3306)/goapm")
	assert.Nil(t, err)
	defer db.Close()

	t.Run("prepare without context should work", func(t *testing.T) {
		stmt, err := db.Prepare("SELECT 1")
		assert.Nil(t, err)
		defer stmt.Close()

		var result string
		err = stmt.QueryRow().Scan(&result)
		assert.Nil(t, err)
		assert.Equal(t, "1", result)
	})

	t.Run("prepare with context should work", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
		defer cancel()

		stmt, err := db.PrepareContext(ctx, "SELECT 1")
		assert.Nil(t, err)
		defer stmt.Close()

		var result string
		err = stmt.QueryRowContext(ctx).Scan(&result)
		assert.Nil(t, err)
		assert.Equal(t, "1", result)
	})

	t.Run("prepare with context and duplicated uid should work(insert and select)", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
		defer cancel()

		uid := uuid.NewString()

		// insert
		stmt, err := db.PrepareContext(ctx, "INSERT INTO `t_user` (`uid`, `name`, `age`, `gender`, `address`, `phone`, `email`, `salary`)"+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?)")
		assert.Nil(t, err)
		defer stmt.Close()

		_, err = stmt.ExecContext(ctx, uid, "John", 18, "male", "Beijing", "1234567890", "john@example.com", 10000)
		assert.Nil(t, err)

		// select
		var result string
		err = db.QueryRowContext(ctx, "SELECT `uid` FROM `t_user` WHERE `uid` = ?", uid).Scan(&result)
		assert.Nil(t, err)
		assert.Equal(t, uid, result)

		// insert again, should duplicate
		_, err = stmt.ExecContext(ctx, uid, "Alice", 18, "female", "Shanghai", "0987654321", "alice@example.com", 20000)
		assert.True(t, strings.Contains(err.Error(), "Duplicate entry"))
	})
}

func Test_SQLDriverWrapper_Transaction(t *testing.T) {
	db, err := NewMySQL("root:root@tcp(127.0.0.1:3306)/goapm")
	assert.Nil(t, err)
	defer db.Close()

	t.Run("transaction without context should work", func(t *testing.T) {
		tx, err := db.Begin()
		assert.Nil(t, err)

		uid := uuid.NewString()
		_, err = tx.Exec("INSERT INTO `t_user` (`uid`, `name`, `age`, `gender`, `address`, `phone`, `email`, `salary`)"+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?)", uid, "John", 18, "male", "Beijing", "1234567890", "john@example.com", 10000)
		assert.Nil(t, err)

		err = tx.Commit()
		assert.Nil(t, err)

		var result string
		err = db.QueryRow("SELECT `uid` FROM `t_user` WHERE `uid` = ?", uid).Scan(&result)
		assert.Nil(t, err)
		assert.Equal(t, uid, result)
	})

	t.Run("transaction with context and commit should work", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
		defer cancel()

		tx, err := db.BeginTx(ctx, nil)
		assert.Nil(t, err)

		uid := uuid.NewString()
		_, err = tx.ExecContext(ctx, "INSERT INTO `t_user` (`uid`, `name`, `age`, `gender`, `address`, `phone`, `email`, `salary`)"+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?)", uid, "John", 18, "male", "Beijing", "1234567890", "john@example.com", 10000)
		assert.Nil(t, err)

		err = tx.Commit()
		assert.Nil(t, err)

		var result string
		err = db.QueryRowContext(ctx, "SELECT `uid` FROM `t_user` WHERE `uid` = ?", uid).Scan(&result)
		assert.Nil(t, err)
		assert.Equal(t, uid, result)
	})

	t.Run("transaction with context and rollback should work", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
		defer cancel()

		tx, err := db.BeginTx(ctx, nil)
		assert.Nil(t, err)

		uid := uuid.NewString()
		_, err = tx.ExecContext(ctx, "INSERT INTO `t_user` (`uid`, `name`, `age`, `gender`, `address`, `phone`, `email`, `salary`)"+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?)", uid, "John", 18, "male", "Beijing", "1234567890", "john@example.com", 10000)
		assert.Nil(t, err)

		err = tx.Rollback()
		assert.Nil(t, err)

		var result string
		err = db.QueryRowContext(ctx, "SELECT `uid` FROM `t_user` WHERE `uid` = ?", uid).Scan(&result)
		assert.Equal(t, sql.ErrNoRows, err)
		assert.Equal(t, "", result)
	})
}
