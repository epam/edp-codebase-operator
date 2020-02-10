package repository

import (
	"database/sql"
	"fmt"
)

const (
	selectPushedValue = "select pushed from \"%v\".codebase where name = $1 ;"
	setPushedValue    = "update \"%v\".codebase set pushed = $1 where name = $2 ;"
)

type CodebaseRepository struct {
	DB *sql.DB
}

func (r CodebaseRepository) SelectPushedValue(name, schema string) (bool, error) {
	stmt, err := r.DB.Prepare(fmt.Sprintf(selectPushedValue, schema))
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	var p *bool
	if err = stmt.QueryRow(name).Scan(&p); err != nil {
		return false, err
	}
	if p == nil {
		return false, nil
	}
	return *p, nil
}

func (r CodebaseRepository) SetPushedValue(pushed bool, name, schema string) error {
	stmt, err := r.DB.Prepare(fmt.Sprintf(setPushedValue, schema))
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(pushed, name)
	return err
}
