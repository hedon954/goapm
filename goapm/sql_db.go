package goapm

import (
	"database/sql"
)

func New(url string) (*sql.DB, error) {
	db, err := sql.Open("mysql", url)
	if err != nil {
		return nil, err
	}

	return db, nil
}
