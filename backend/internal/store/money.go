package store

import (
	"fmt"

	"pvmoney/internal/models"
)

func CheckMoney(n int64) error {
	if n < 0 {
		return fmt.Errorf("%w: amount must be >= 0", ErrInvalid)
	}
	if n > models.MaxMoney {
		return fmt.Errorf("%w: amount too large", ErrInvalid)
	}
	return nil
}

func CheckMoneyPositive(n int64) error {
	if n <= 0 {
		return fmt.Errorf("%w: amount must be positive", ErrInvalid)
	}
	if n > models.MaxMoney {
		return fmt.Errorf("%w: amount too large", ErrInvalid)
	}
	return nil
}
