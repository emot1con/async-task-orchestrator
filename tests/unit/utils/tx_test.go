package utils_test

import (
	"database/sql"
	"errors"
	"task_handler/internal/utils"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/lib/pq"
	"github.com/DATA-DOG/go-sqlmock"
)

func TestWithTransaction_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectCommit()

	called := false
	err = utils.WithTransaction(db, func(tx *sql.Tx) error {
		called = true
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, called)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWithTransaction_FnError_Rollback(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectRollback()

	err = utils.WithTransaction(db, func(tx *sql.Tx) error {
		return errors.New("something failed")
	})

	assert.Error(t, err)
	assert.EqualError(t, err, "something failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWithTransaction_BeginError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin().WillReturnError(errors.New("cannot start transaction"))

	err = utils.WithTransaction(db, func(tx *sql.Tx) error {
		return nil
	})

	assert.Error(t, err)
	assert.EqualError(t, err, "cannot start transaction")
}
