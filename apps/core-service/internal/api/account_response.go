package api

import (
	db "github.com/DewaSRY/core-service/internal/db/sqlc"
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

type accountuserResponse struct {
	ID          int64  `json:"id"`
	Balance     string `json:"balance"`
	Currency    string `json:"currency"`
	UserID      int64  `json:"user_id"`
	Number      string `json:"number"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsMain      bool   `json:"is_main"`
	CreatedAt   string `json:"created_at"`
	Username    string `json:"username"`
}

// toAccountuserResponse is the single mapper for every query that selects
// account_user_details_view via sqlc.embed(v) — ListMeAccountsByUserId,
// ListAccountsSearchByUserNumber, and GetAccountViewById all produce a Row
// type wrapping db.AccountUserDetailsView, so a new query against this view
// never needs its own mapper.
func toAccountuserResponse(account db.AccountUserDetailsView) accountuserResponse {
	return accountuserResponse{
		ID:          account.ID,
		Balance:     account.Balance,
		Currency:    account.Currency,
		UserID:      account.UserID.Int64,
		Number:      account.Number.String,
		Name:        account.Name.String,
		Description: account.Description.String,
		IsMain:      account.IsMain,
		CreatedAt:   account.CreatedAt.Format("2006-01-02 15:04:05"),
		Username:    account.Username.String,
	}
}

// publicAccountResponse is the minimal, non-sensitive view of an account
// used when the account being looked up may belong to another user (transfer
// destination search, recent destinations) — it never exposes balance or
// user_id.
type publicAccountResponse struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Number   string `json:"number"`
	Username string `json:"username"`
	UserID   int64  `json:"user_id"`
}

func toPublicAccountResponseFromDestination(account db.ListRecentTransferDestinationsRow) publicAccountResponse {
	return publicAccountResponse{
		ID:       account.ID,
		Name:     account.Name.String,
		Number:   account.Number.String,
		Username: account.Username,
		UserID:   account.UserID,
	}
}

type accountEntriesViewResponse struct {
	ID        int64  `json:"id"`
	AccountID int64  `json:"account_id"`
	Amount    string `json:"amount"`

	AccountName     string `json:"account_name"`
	AccountNumber   string `json:"account_number"`
	IsMain          bool   `json:"is_main"`
	ToAccountName   string `json:"to_account_name"`
	ToAccountNumber string `json:"to_account_number"`
	UserID          int64  `json:"user_id"`
	ToAccountID     int64  `json:"to_account_id"`
	Type            string `json:"type"`
	Description     string `json:"description"`
}

// toAccountEntriesViewResponse is the single mapper for every query that
// selects account_entries_view via sqlc.embed(v) — AccountEntriesByAccountId
// and ListAccountEntriesByAccountId both produce a Row type wrapping
// db.AccountEntriesView.
func toAccountEntriesViewResponse(entry db.AccountEntriesView) accountEntriesViewResponse {
	return accountEntriesViewResponse{
		ID:              entry.ID,
		AccountID:       entry.AccountID,
		Amount:          entry.Amount,
		AccountName:     entry.AccountName.String,
		AccountNumber:   entry.AccountNumber.String,
		IsMain:          entry.IsMain.Bool,
		ToAccountName:   entry.ToAccountName.String,
		ToAccountNumber: entry.ToAccountNumber.String,
		UserID:          entry.UserID.Int64,
		ToAccountID:     entry.ToAccountID.Int64,
		Type:            entry.Type,
		Description:     entry.Description.String,
	}
}

type transactionHistoryCounterparty struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Number string `json:"number"`
}

type transactionHistoryItem struct {
	ID           int64                           `json:"id"`
	Label        string                          `json:"label"`
	Amount       string                          `json:"amount"`
	Currency     string                          `json:"currency"`
	Description  string                          `json:"description"`
	CreatedAt    string                          `json:"created_at"`
	Counterparty *transactionHistoryCounterparty `json:"counterparty"`
}
