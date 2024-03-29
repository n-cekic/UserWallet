package transactions

import (
	"context"
	"database/sql"
	"errors"
	"time"
	L "userTransactions/logging"

	_ "github.com/lib/pq"
)

type Repo struct {
	db *sql.DB
}

func (r *Repo) getBalance(id int) float64 {
	var balance float64
	q := "SELECT balance FROM userbalance WHERE user_id = $1"
	err := r.db.QueryRow(q, id).Scan(&balance)
	if err != nil {
		L.Logger.Error("Failed getting user ", id, " balance", err)
		return 0
	}

	return balance
}

func (r *Repo) deposit(userId *int, amount float64) (float64, error) {
	if userId == nil {
		return 0, errors.New("user ID is required")
	}

	if amount <= 0 {
		return 0, errors.New("amount must be positive number")
	}

	tx, err := r.db.BeginTx(context.Background(), nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var currentBalance float64
	q := "SELECT balance FROM userbalance WHERE user_id = $1"
	err = tx.QueryRow(q, userId).Scan(&currentBalance)
	if err != nil {
		return 0, err
	}

	newBalance := currentBalance + amount
	q = "UPDATE userbalance SET balance = $1 WHERE user_id = $2"
	tx.Exec(q, newBalance, userId)
	if err != nil {
		return 0, err
	}

	err = r.newTransactionRecord(tx, userId, userId, "D", amount)

	L.Logger.Info("Deposited successfully")
	tx.Commit()
	return newBalance, err
}

func (r *Repo) transfer(senderId, recipientId *int, amount float64) error {
	if senderId == nil {
		return errors.New("sender ID is required")
	}

	if amount <= 0 {
		return errors.New("amount must be positive number")
	}

	if recipientId == nil {
		return errors.New("recipient is required to execute transaction")
	}

	if *recipientId == *senderId {
		return errors.New("recipient and sender IDs must be different from one another")
	}

	tx, err := r.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var currentBalance float64
	q := "SELECT balance FROM userbalance WHERE user_id = $1"
	err = tx.QueryRow(q, senderId).Scan(&currentBalance)
	if err != nil {
		return err
	}

	if currentBalance-amount < 0 {
		return errors.New("insufficient funds on the account")
	}

	newBalance := currentBalance - amount
	q = "UPDATE userbalance SET balance = $1 WHERE user_id = $2"
	_, err = tx.Exec(q, newBalance, senderId)
	if err != nil {
		return err
	}

	q = "SELECT balance FROM userbalance WHERE user_id = $1"
	var currentBalanceRecipient float64
	err = tx.QueryRow(q, recipientId).Scan(&currentBalanceRecipient)
	if err != nil {
		return err
	}

	newBalance = currentBalanceRecipient + amount
	q = "UPDATE userbalance SET balance = $1 WHERE user_id = $2"
	_, err = tx.Exec(q, newBalance, recipientId)
	if err != nil {
		return err
	}

	err = r.newTransactionRecord(tx, senderId, recipientId, "T", amount)
	if err != nil {
		L.Logger.Error("Failed to record the transaction: ", err)
		return err
	}

	L.Logger.Info("Transfer finished successfully")
	tx.Commit()
	return nil
}

func (r *Repo) newTransactionRecord(tx *sql.Tx, from, to *int, operation string, amount float64) error {
	q := `INSERT INTO transaction ("from", "to", "type", amount) VALUES ($1, $2, $3, $4)`

	_, err := tx.Exec(q, from, to, operation, amount)
	return err
}

func (r *Repo) insertNewUser(userId int, createdAt time.Time) error {
	q := `INSERT INTO userbalance (user_id, balance, created_at) VALUES ($1, $2, $3)`
	_, err := r.db.Exec(q, userId, 0, createdAt)
	return err
}
