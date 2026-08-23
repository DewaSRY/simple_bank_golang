package mapper

import (
	sqlc "github.com/DewaSRY/core-service/db/sqlc"
)

func UpdateBalanceAccountToAccount(updateAccount sqlc.UpdateAccountBalanceRow) sqlc.Account {
	return sqlc.Account{
		ID:        updateAccount.ID,
		Owner:     updateAccount.Owner,
		Balance:   updateAccount.Balance,
		Currency:  updateAccount.Currency,
		CreatedAt: updateAccount.CreatedAt,
	}
}
