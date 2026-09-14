package mapper

import (
	sqlc "github.com/DewaSRY/core-service/internal/db/sqlc"
)

func CreateEntriesRowToEntry(row sqlc.CreateEntriesRow) sqlc.Entry {
	return sqlc.Entry{
		ID:          row.ID,
		AccountID:   row.AccountID,
		Type:        row.Type,
		Amount:      row.Amount,
		Description: row.Description,
		TransferID:  row.TransferID,
		CreatedAt:   row.CreatedAt,
	}
}
