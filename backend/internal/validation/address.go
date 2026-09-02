package validation

import (
	"fmt"
	"regexp"
	"strings"
)

var hexAddressRe = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

// ErrInvalidAddress is returned when an Ethereum address fails validation.
type ErrInvalidAddress struct {
	Input string
}

func (e ErrInvalidAddress) Error() string {
	return fmt.Sprintf("invalid Ethereum address: %q", e.Input)
}

// ValidateAddress checks that the given string is a well-formed Ethereum
// address. It is intentionally permissive about checksum casing — callers
// that need EIP-55 checksumming should apply it separately.
func ValidateAddress(addr string) error {
	if addr == "" {
		return ErrInvalidAddress{Input: addr}
	}
	if !hexAddressRe.MatchString(addr) {
		return ErrInvalidAddress{Input: addr}
	}
	return nil
}

// NormalizeAddress lower-cases the hex portion of an Ethereum address so
// comparisons and database lookups are consistent.
func NormalizeAddress(addr string) string {
	return strings.ToLower(addr)
}

// ValidatePaginationParams returns sanitised page/perPage values.
// page is 1-indexed. perPage is clamped to [1, 100].
func ValidatePaginationParams(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}
