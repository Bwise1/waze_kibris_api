package util

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/bwise1/waze_kibris/internal/model"
)

// randomIndex returns a cryptographically secure random index in [0, max).
// Falls back to 0 on error, which is acceptable for non-critical pseudonyms.
func randomIndex(max int64) int64 {
	if max <= 0 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 0
	}
	return n.Int64()
}

// GenerateDisplayName creates a pseudonymous, driver-themed username in the form:
//   Adjective + Noun + 3-digit number
// Example: "SwiftDriver742".
func GenerateDisplayName() string {
	adj := model.Adjectives[randomIndex(int64(len(model.Adjectives)))]
	noun := model.Nouns[randomIndex(int64(len(model.Nouns)))]

	// 3-digit suffix in [100, 999]
	num := randomIndex(900) + 100

	return fmt.Sprintf("%s%s%d", adj, noun, num)
}

