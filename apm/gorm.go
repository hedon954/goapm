package apm

import (
	"context"
	"database/sql"
	"fmt"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// NewGorm returns a new Gorm DB with hooks.
func NewGorm(name, connectURL string) (*gorm.DB, error) {
	db, err := gorm.Open(newGormDialector(name, connectURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	Logger.Info(context.TODO(), fmt.Sprintf("mysql gorm client[%s] connected", name), nil)
	return db, nil
}

// gormDialector is a wrapper of gorm.Dialector which provides hooks.
type gormDialector struct {
	connectURL string
	driverName string
	gorm.Dialector
}

func newGormDialector(name, connectURL string) *gormDialector {
	driverName := fmt.Sprintf("%s-%s", "mysql-wrapper", uuid.NewString())
	sql.Register(driverName, wrap(&mysqldriver.MySQLDriver{}, name, connectURL))
	return &gormDialector{
		connectURL: connectURL,
		driverName: driverName,
		Dialector:  mysql.Open(connectURL),
	}
}

func (d *gormDialector) Name() string {
	return d.driverName
}
