package store

import (
	"context"
	"database/sql"

	mapper "github.com/DewaSRY/core-service/db/mapper"
	sqlc "github.com/DewaSRY/core-service/db/sqlc"
)

type CreateAccountTxParams struct {
	UserID      sql.NullInt64
	Name        string
	Description string
	IsMain      bool
}

// CreateAccountTx creates an account and assigns it a globally-unique number
// derived from its own id, which is only known after the insert. The insert
// and the follow-up number update happen in one DB transaction so an account
// can never be left with a permanently-NULL number.
func (store *_store) CreateAccountTx(ctx context.Context, arg CreateAccountTxParams) (sqlc.Account, error) {
	var result sqlc.Account

	err := store.execTx(ctx, func(q sqlc.Querier) error {
		var err error
		result, err = createAccountTx(ctx, q, arg)
		return err
	})

	return result, err
}

func createAccountTx(ctx context.Context, q sqlc.Querier, arg CreateAccountTxParams) (sqlc.Account, error) {
	inserted, err := q.CreateAccount(ctx, sqlc.CreateAccountParams{
		Name:        sql.NullString{String: arg.Name, Valid: arg.Name != ""},
		Description: sql.NullString{String: arg.Description, Valid: arg.Description != ""},
		Balance:     "0",
		Currency:    "IDR",
		UserID:      arg.UserID,
		IsMain:      arg.IsMain,
	})
	if err != nil {
		return sqlc.Account{}, err
	}

	updated, err := q.UpdateAccountNumber(ctx, sqlc.UpdateAccountNumberParams{
		ID:     inserted.ID,
		Number: sql.NullString{String: generateAccountNumber(inserted.ID), Valid: true},
	})

	if err != nil {
		return sqlc.Account{}, err
	}

	return mapper.UpdateAccountNumberRowToAccount(updated), nil
}
