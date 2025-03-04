package apm

import (
	"os"
	"testing"

	"github.com/hedon954/goapm/internal/testutils"
)

var (
	mysqlDSN string
	redisDSN string
)

func TestMain(m *testing.M) {
	var mysqlShutdown func()
	var redisShutdown func()
	mysqlDSN, mysqlShutdown = testutils.PrepareMySQL(&User{})
	redisDSN, redisShutdown = testutils.PrepareRedis()
	os.Exit(func() int {
		defer mysqlShutdown()
		defer redisShutdown()
		return m.Run()
	}())
}

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
