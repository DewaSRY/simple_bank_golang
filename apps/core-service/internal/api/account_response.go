package api

import (
	mapper "github.com/DewaSRY/core-service/db/mapper"
	db "github.com/DewaSRY/core-service/db/sqlc"
)

type accountResponse struct {
	ID          int64  `json:"id"`
	Balance     string `json:"balance"`
	Currency    string `json:"currency"`
	UserID      int64  `json:"user_id"`
	Number      string `json:"number"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsMain      bool   `json:"is_main"`
	CreatedAt   string `json:"created_at"`
}

func toAccountResponse(account db.Account) accountResponse {
	return accountResponse{
		ID:          account.ID,
		Balance:     account.Balance,
		Currency:    account.Currency,
		UserID:      account.UserID.Int64,
		Number:      account.Number.String,
		Name:        account.Name.String,
		Description: account.Description.String,
		IsMain:      account.IsMain,
		CreatedAt:   account.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func toListAccountResponse(accounts []db.ListAccountsByUserIdRow) []accountResponse {
	accountResponses := make([]accountResponse, len(accounts))
	for i, account := range accounts {
		accountResponses[i] = toAccountResponse(mapper.ListAccountsByUserIdRowToAccount(account))
	}
	return accountResponses
}

// publicAccountResponse is the minimal, non-sensitive view of an account
// used when the account being looked up may belong to another user (transfer
// destination search, recent destinations) — it never exposes balance or
// user_id.
type publicAccountResponse struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Number string `json:"number"`
}
