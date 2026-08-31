package api

import (
	db "github.com/DewaSRY/core-service/db/sqlc"
)

type accountResponse struct {
	ID        int64  `json:"id"`
	Balance   string `json:"balance"`
	Currency  string `json:"currency"`
	UserID    int64  `json:"user_id"`
	CreatedAt string `json:"created_at"`
}

func toAccountResponse(account db.ListAccountsByUserIdRow) accountResponse {
	return accountResponse{
		ID:        account.ID,
		Balance:   account.Balance,
		Currency:  account.Currency,
		UserID:    account.UserID.Int64,
		CreatedAt: account.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func toListAccountResponse(accounts []db.ListAccountsByUserIdRow) []accountResponse {
	accountResponses := make([]accountResponse, len(accounts))
	for i, account := range accounts {
		accountResponses[i] = toAccountResponse(account)
	}
	return accountResponses
}
