package tests

import (
	"testing"

	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/validation"
)

func TestValidateAddress(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid lowercase",  "0xd8da6bf26964af9d7eed9e03e53415d37aa96045", false},
		{"valid uppercase",  "0xD8DA6BF26964AF9D7EED9E03E53415D37AA96045", false},
		{"valid mixed case", "0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAE", false},
		{"empty string",     "", true},
		{"no 0x prefix",     "d8da6bf26964af9d7eed9e03e53415d37aa96045", true},
		{"too short",        "0xd8da6bf26964af9d7eed9e03e53415d37aa960", true},
		{"too long",         "0xd8da6bf26964af9d7eed9e03e53415d37aa960450000", true},
		{"invalid hex char", "0xd8da6bf26964af9d7eed9e03e53415d37aa9604g", true},
		{"just 0x",          "0x", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateAddress(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAddress(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeAddress(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"0xD8DA6BF26964AF9D7EED9E03E53415D37AA96045", "0xd8da6bf26964af9d7eed9e03e53415d37aa96045"},
		{"0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAE", "0xde0b295669a9fd93d5f28d9ec85e40f4cb697bae"},
		{"0xabcdef", "0xabcdef"},
	}
	for _, c := range cases {
		got := validation.NormalizeAddress(c.input)
		if got != c.want {
			t.Errorf("NormalizeAddress(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestValidatePaginationParams(t *testing.T) {
	tests := []struct {
		page, perPage       int
		wantPage, wantPer   int
	}{
		{1, 20, 1, 20},
		{0, 0, 1, 20},   // defaults applied
		{-1, -5, 1, 20}, // clamped to minimums
		{5, 200, 5, 100}, // perPage clamped to max
		{3, 50, 3, 50},
	}
	for _, tt := range tests {
		gotPage, gotPer := validation.ValidatePaginationParams(tt.page, tt.perPage)
		if gotPage != tt.wantPage || gotPer != tt.wantPer {
			t.Errorf("ValidatePaginationParams(%d, %d) = (%d, %d), want (%d, %d)",
				tt.page, tt.perPage, gotPage, gotPer, tt.wantPage, tt.wantPer)
		}
	}
}
