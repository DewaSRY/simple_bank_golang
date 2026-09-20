package account

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
// account_user_details_view via sqlc.embed(v) — ListMeAccountsByUserId and
// GetAccountViewById produce a Row type wrapping db.AccountUserDetailsView,
// so a new query against this view never needs its own mapper. Only used
// where the caller is known to own the account: it exposes balance and
// description, so ListAccountsSearchByUserNumber (which can match any
// user's account) must use accountSearchResponse instead.
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

// accountSearchResponse is the minimal, non-sensitive view returned for a
// transfer-destination lookup by account number — it never exposes another
// user's balance or description, matching the shape already used by
// listRecentTransferDestinations in the transfer package.
type accountSearchResponse struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Number   string `json:"number"`
	Username string `json:"username"`
	UserID   int64  `json:"user_id"`
}

func toAccountSearchResponse(account db.AccountUserDetailsView) accountSearchResponse {
	return accountSearchResponse{
		ID:       account.ID,
		Name:     account.Name.String,
		Number:   account.Number.String,
		Username: account.Username.String,
		UserID:   account.UserID.Int64,
	}
}
