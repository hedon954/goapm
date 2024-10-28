package goapm

import (
	"database/sql"
	"fmt"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// NewGorm returns a new Gorm DB with hooks.
func NewGorm(connectURL string) (*gorm.DB, error) {
	db, err := gorm.Open(newGormDialector(connectURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

// gormDialector is a wrapper of gorm.Dialector which provides hooks.
type gormDialector struct {
	connectURL string
	driverName string
	gorm.Dialector
}

func newGormDialector(connectURL string) *gormDialector {
	driverName := fmt.Sprintf("%s-%s", "mysql-wrapper", uuid.NewString())
	sql.Register(driverName, wrap(&mysqldriver.MySQLDriver{}, connectURL))
	return &gormDialector{
		connectURL: connectURL,
		driverName: driverName,
		Dialector:  mysql.Open(connectURL),
	}
}

func (d *gormDialector) Name() string {
	return d.driverName
}
