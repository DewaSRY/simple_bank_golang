package store

import "errors"

var (
	ErrSameAccount             = errors.New("from_account_id and to_account_id must be different")
	ErrCurrencyMismatch        = errors.New("from and to accounts must share the same currency")
	ErrInvalidAmount           = errors.New("amount must be greater than zero")
	ErrInsufficientFunds       = errors.New("account does not have sufficient funds for this transfer")
	ErrCannotDeleteMainAccount = errors.New("main account cannot be deleted")
)
