package money

import "errors"

// Amount represents monetary value in the smallest currency unit (e.g., paise, cents)
type Amount int64

// Common errors
var (
	ErrNegativeAmount = errors.New("amount cannot be negative")
	ErrZeroAmount     = errors.New("amount must be greater than zero")
)

// New creates a new Amount, validating it's positive
func New(val int64) (Amount, error) {
	if val < 0 {
		return 0, ErrNegativeAmount
	}
	return Amount(val), nil
}

// Add adds two amounts safely
func (a Amount) Add(b Amount) Amount {
	return a + b
}

// Subtract subtracts two amounts, returning an error if the result would be negative
func (a Amount) Subtract(b Amount) (Amount, error) {
	if a < b {
		return 0, ErrNegativeAmount
	}
	return a - b, nil
}

// Float64 converts to a decimal representation (for display purposes ONLY)
func (a Amount) Float64() float64 {
	return float64(a) / 100.0
}
