package api

import db "github.com/DewaSRY/core-service/db/sqlc"

func toTransactionHistoryItem(row db.ListAccountTransactionHistoryRow) transactionHistoryItem {
	item := transactionHistoryItem{
		ID:          row.ID,
		Amount:      row.Amount,
		Currency:    "IDR",
		Description: row.Description.String,
		CreatedAt:   row.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	switch row.Type {
	case "SEND":
		item.Label = "Transfer Out"
	case "RECEIVED":
		item.Label = "Transfer In"
	default:
		item.Label = "Deposit"
	}

	if row.CounterpartyAccountID.Valid {
		item.Counterparty = &transactionHistoryCounterparty{
			ID:     row.CounterpartyAccountID.Int64,
			Name:   row.CounterpartyAccountName.String,
			Number: row.CounterpartyAccountNumber.String,
		}
	}

	return item
}
