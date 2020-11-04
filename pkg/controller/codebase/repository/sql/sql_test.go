package sql

import (
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

func NewMock() (*CodebaseRepository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		log.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	return &CodebaseRepository{DB: db}, mock
}

const (
	expStatus = "status-quo"
	codebase  = "super-codebase"
	edp       = "super-edp"

	edpSelectQuery = "select project_status from \"super-edp\"\\.codebase where name = \\$1"
	edpUpdateQuery = "update \"super-edp\"\\.codebase set project_status = \\$1 where name = \\$2 ;"
)

func TestSqlCodebaseRepository_SelectProjectStatusValue(t *testing.T) {
	sqlRepo, mock := NewMock()
	rows := sqlmock.NewRows([]string{"project_status"}).AddRow(expStatus)
	prep := mock.ExpectPrepare(edpSelectQuery)
	prep.ExpectQuery().WithArgs(codebase).WillReturnRows(rows)

	st, err := sqlRepo.SelectProjectStatusValue(codebase, edp)

	assert.NoError(t, err)
	assert.Equal(t, expStatus, *st)
}

func TestSqlCodebaseRepository_SelectProjectStatusValueError(t *testing.T) {
	sqlRepo, mock := NewMock()
	prep := mock.ExpectPrepare(edpSelectQuery)
	prep.ExpectQuery().WithArgs(codebase).WillReturnError(errors.New("some sql error"))

	st, err := sqlRepo.SelectProjectStatusValue(codebase, edp)

	assert.Error(t, err)
	assert.Nil(t, st)
}

func TestSqlCodebaseRepository_SelectProjectStatusValueErrorPrepare(t *testing.T) {
	sqlRepo, mock := NewMock()
	prep := mock.ExpectPrepare(edpSelectQuery)
	prep.WillReturnError(errors.New("some prepare error"))

	st, err := sqlRepo.SelectProjectStatusValue(codebase, edp)

	assert.Error(t, err)
	assert.Nil(t, st)
}

func TestSqlCodebaseRepository_UpdateProjectStatusValue(t *testing.T) {
	sqlRepo, mock := NewMock()
	prep := mock.ExpectPrepare(edpUpdateQuery)
	prep.ExpectExec().WithArgs(expStatus, codebase).WillReturnResult(sqlmock.NewResult(0, 1))

	err := sqlRepo.UpdateProjectStatusValue(expStatus, codebase, edp)

	assert.NoError(t, err)
}

func TestSqlCodebaseRepository_UpdateProjectStatusValueErrorPrepare(t *testing.T) {
	sqlRepo, mock := NewMock()
	prep := mock.ExpectPrepare(edpUpdateQuery)
	prep.WillReturnError(errors.New("some prepare error"))

	err := sqlRepo.UpdateProjectStatusValue(expStatus, codebase, edp)

	assert.Error(t, err)
}
