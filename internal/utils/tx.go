package utils

import "github.com/sirupsen/logrus"

import "database/sql"

func WithTransaction(db *sql.DB, fn func(tx *sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	logrus.Info("Transaction started")

	defer func() {
		if r := recover(); r != nil {
			logrus.Info("Panic occurred, rolling back transaction")
			_ = tx.Rollback()
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		logrus.Info("Error occurred, rolling back transaction")
		return err
	}

	logrus.Info("Transaction committed successfully")
	return tx.Commit()
}
