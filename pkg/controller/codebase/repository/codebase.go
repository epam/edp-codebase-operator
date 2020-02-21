package repository

import (
	"database/sql"
	"fmt"
)

const (
	selectProjectStatusValue = "select project_status from \"%v\".codebase where name = $1 ;"
	setProjectStatusValue    = "update \"%v\".codebase set project_status = $1 where name = $2 ;"
)

type CodebaseRepository struct {
	DB *sql.DB
}

func (r CodebaseRepository) SelectProjectStatusValue(name, schema string) (*string, error) {
	stmt, err := r.DB.Prepare(fmt.Sprintf(selectProjectStatusValue, schema))
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var s *string
	if err = stmt.QueryRow(name).Scan(&s); err != nil {
		return nil, err
	}
	return s, nil
}

func (r CodebaseRepository) UpdateProjectStatusValue(status, name, schema string) error {
	stmt, err := r.DB.Prepare(fmt.Sprintf(setProjectStatusValue, schema))
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(status, name)
	return err
}
