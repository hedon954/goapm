package testutils

import (
	"fmt"

	"github.com/hedon954/go-mysql-mocker/gmm"
	"gorm.io/gorm/schema"
)

func PrepareMySQL(models ...schema.Tabler) (dsn string, shutdown func()) {
	gmmBuilder := gmm.Builder("goapm")
	for _, model := range models {
		_ = gmmBuilder.CreateTable(model)
	}
	_, _, shutdown, err := gmmBuilder.Build()
	if err != nil {
		panic(err)
	}

	dsn = fmt.Sprintf("root:root@tcp(127.0.0.1:%d)/goapm", gmmBuilder.GetPort())
	return
}
