package mapper

import (
	sqlc "github.com/DewaSRY/core-service/db/sqlc"
)

func UpdateBalanceAccountToAccount(updateAccount sqlc.IncrementAccountBalanceRow) sqlc.Account {
	return sqlc.Account{
		ID:          updateAccount.ID,
		Balance:     updateAccount.Balance,
		Currency:    updateAccount.Currency,
		UserID:      updateAccount.UserID,
		Number:      updateAccount.Number,
		Name:        updateAccount.Name,
		Description: updateAccount.Description,
		IsMain:      updateAccount.IsMain,
		CreatedAt:   updateAccount.CreatedAt,
	}
}

func UpdateAccountNumberRowToAccount(row sqlc.UpdateAccountNumberRow) sqlc.Account {
	return sqlc.Account{
		ID:          row.ID,
		Balance:     row.Balance,
		Currency:    row.Currency,
		UserID:      row.UserID,
		Number:      row.Number,
		Name:        row.Name,
		Description: row.Description,
		IsMain:      row.IsMain,
		CreatedAt:   row.CreatedAt,
	}
}

func UpdateAccountRowToAccount(row sqlc.UpdateAccountRow) sqlc.Account {
	return sqlc.Account{
		ID:          row.ID,
		Balance:     row.Balance,
		Currency:    row.Currency,
		UserID:      row.UserID,
		Number:      row.Number,
		Name:        row.Name,
		Description: row.Description,
		IsMain:      row.IsMain,
		CreatedAt:   row.CreatedAt,
	}
}

func SoftDeleteAccountRowToAccount(row sqlc.SoftDeleteAccountRow) sqlc.Account {
	return sqlc.Account{
		ID:          row.ID,
		Balance:     row.Balance,
		Currency:    row.Currency,
		UserID:      row.UserID,
		Number:      row.Number,
		Name:        row.Name,
		Description: row.Description,
		IsMain:      row.IsMain,
		CreatedAt:   row.CreatedAt,
	}
}

func GetMainAccountByUserIdRowToAccount(row sqlc.GetMainAccountByUserIdRow) sqlc.Account {
	return sqlc.Account{
		ID:          row.ID,
		Balance:     row.Balance,
		Currency:    row.Currency,
		UserID:      row.UserID,
		Number:      row.Number,
		Name:        row.Name,
		Description: row.Description,
		IsMain:      row.IsMain,
		CreatedAt:   row.CreatedAt,
	}
}

func GetAccountByIdRowToAccount(row sqlc.GetAccountByIdRow) sqlc.Account {
	return sqlc.Account{
		ID:          row.ID,
		Balance:     row.Balance,
		Currency:    row.Currency,
		UserID:      row.UserID,
		Number:      row.Number,
		Name:        row.Name,
		Description: row.Description,
		IsMain:      row.IsMain,
		CreatedAt:   row.CreatedAt,
	}
}

func ListAccountsByUserIdRowToAccount(row sqlc.ListAccountsByUserIdRow) sqlc.Account {
	return sqlc.Account{
		ID:          row.ID,
		Balance:     row.Balance,
		Currency:    row.Currency,
		UserID:      row.UserID,
		Number:      row.Number,
		Name:        row.Name,
		Description: row.Description,
		IsMain:      row.IsMain,
		CreatedAt:   row.CreatedAt,
	}
}
