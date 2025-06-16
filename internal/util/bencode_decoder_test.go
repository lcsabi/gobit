package util

import (
	"bytes"
	"testing"
)

func TestParseInteger(t *testing.T) {
	testCases := []struct {
		input    string
		expected int64
	}{
		{"i0e", 0},
		{"i10e", 10},
		{"i-1e", -1},
	}

	for _, tc := range testCases {
		got, err := parseInt(bytes.NewReader([]byte(tc.input[1:])))
		if err != nil {
			t.Errorf("parseInt(%q) returned error: %v", tc.input, err)
			continue
		}

		if got != tc.expected {
			t.Errorf("parseInt(%q) => got: %v, want: %d", tc.input, got, tc.expected)
		}
	}
}

func TestParseString(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"4:spam", "spam"},
		{"0:", ""},
		{"12:spamspamspam", "spamspamspam"},
	}

	for _, tc := range testCases {
		got, err := parseByteString(bytes.NewReader([]byte(tc.input[1:])), tc.input[0])
		if err != nil {
			t.Errorf("parseByteString(%q) returned error: %v", tc.input, err)
			continue
		}

		if got != tc.expected {
			t.Errorf("parseByteString(%q) => got: %v, want: %s", tc.input, got, tc.expected)
		}
	}
}
