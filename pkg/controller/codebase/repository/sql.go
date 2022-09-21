package repository

import (
	"database/sql"
	"fmt"

	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

const (
	selectProjectStatusValue = "select project_status from \"%v\".codebase where name = $1 ;"
	setProjectStatusValue    = "update \"%v\".codebase set project_status = $1 where name = $2 ;"
)

type SqlCodebaseRepository struct {
	DB *sql.DB
}

func (r SqlCodebaseRepository) SelectProjectStatusValue(name, schema string) (val string, err error) {
	stmt, err := r.DB.Prepare(fmt.Sprintf(selectProjectStatusValue, schema))
	if err != nil {
		return "", err
	}

	defer util.CloseWithErrorCapture(&err, stmt, "failed to close SQL prepared statement")

	err = stmt.QueryRow(name).Scan(&val)
	if err != nil {
		return "", err
	}

	return
}

func (r SqlCodebaseRepository) UpdateProjectStatusValue(status, name, schema string) (err error) {
	stmt, err := r.DB.Prepare(fmt.Sprintf(setProjectStatusValue, schema))
	if err != nil {
		return err
	}

	defer util.CloseWithErrorCapture(&err, stmt, "failed to close SQL prepared statement")

	_, err = stmt.Exec(status, name)
	return err
}
