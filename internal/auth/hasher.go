package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidPassword = errors.New("invalid password")
	ErrPasswordTooLong = errors.New("password exceeds maximum allowed length")
)

const (
	// DefaultCost sets the bcrypt computational cost
	DefaultCost = 12
	// MaxPasswordLength limits password input to prevent DoS via excessive bcrypt hashing
	MaxPasswordLength = 72
)

// PasswordHasher defines password hashing and verification contract
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hashedPassword, password string) error
}

// BcryptHasher implements PasswordHasher using standard bcrypt
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher creates a new BcryptHasher
func NewBcryptHasher(cost ...int) *BcryptHasher {
	c := DefaultCost
	if len(cost) > 0 && cost[0] >= bcrypt.MinCost && cost[0] <= bcrypt.MaxCost {
		c = cost[0]
	}
	return &BcryptHasher{cost: c}
}

// Hash generates a secure bcrypt hash of the given plaintext password
func (h *BcryptHasher) Hash(password string) (string, error) {
	if len(password) > MaxPasswordLength {
		return "", ErrPasswordTooLong
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

// Compare checks whether the plaintext password matches the hashed password
func (h *BcryptHasher) Compare(hashedPassword, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrInvalidPassword
		}
		return fmt.Errorf("password comparison failed: %w", err)
	}
	return nil
}
