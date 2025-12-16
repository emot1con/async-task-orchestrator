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

// func StartWorker(conn *amqp.Connection, workerID int) {
//     ch, err := CreateChannel(conn)
//     if err != nil {
//         log.Fatalf("worker %d failed to create channel: %v", workerID, err)
//     }
//     defer ch.Close()

//     ch.Qos(1, 0, false)

//     msgs, err := ch.Consume(
//         "task_queue",
//         "",
//         false, // manual ACK
//         false,
//         false,
//         false,
//         nil,
//     )
//     if err != nil {
//         log.Fatalf("worker %d failed to consume: %v", workerID, err)
//     }

//     log.Printf("Worker %d started", workerID)

//     for msg := range msgs {
//         log.Printf("Worker %d received: %s", workerID, msg.Body)

//         // simulasi proses
//         time.Sleep(2 * time.Second)

//         msg.Ack(false)
//     }
// }
