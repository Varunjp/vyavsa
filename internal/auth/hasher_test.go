package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestBcryptHasher(t *testing.T) {
	hasher := NewBcryptHasher(bcrypt.MinCost) // MinCost for fast unit tests

	t.Run("Successfully hashes and validates password", func(t *testing.T) {
		password := "SecretPass123!"
		hash, err := hasher.Hash(password)
		assert.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, password, hash)

		// Successful compare
		err = hasher.Compare(hash, password)
		assert.NoError(t, err)

		// Failed compare
		err = hasher.Compare(hash, "WrongPassword")
		assert.ErrorIs(t, err, ErrInvalidPassword)
	})

	t.Run("Rejects passwords exceeding 72 characters", func(t *testing.T) {
		tooLong := strings.Repeat("a", 73)
		_, err := hasher.Hash(tooLong)
		assert.ErrorIs(t, err, ErrPasswordTooLong)
	})
}
