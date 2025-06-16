package util

import (
	"bytes"
	"reflect"
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
		got, err := decodeInteger(bytes.NewReader([]byte(tc.input[1:]))) // skip 'i'
		if err != nil {
			t.Errorf("parseInt(%q) returned error: %v", tc.input, err)
			continue
		}

		if got != tc.expected {
			t.Errorf("parseInt(%q) => got: %v want: %d", tc.input, got, tc.expected)
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
		got, err := decodeByteString(bytes.NewReader([]byte(tc.input[1:])), tc.input[0]) // skip first digit
		if err != nil {
			t.Errorf("parseByteString(%q) returned error: %v", tc.input, err)
			continue
		}

		if got != tc.expected {
			t.Errorf("parseByteString(%q) => got: %v want: %s", tc.input, got, tc.expected)
		}
	}
}

func TestParseList(t *testing.T) {
	testCases := []struct {
		input    string
		expected []BencodedValue
	}{
		{"l4:spam4:eggse", []BencodedValue{"spam", "eggs"}},
		{"le", nil},
		{"li1ei20e4:spame", []BencodedValue{int64(1), int64(20), "spam"}},
		{"l3:mooi42ee", []BencodedValue{"moo", int64(42)}},
	}

	for _, tc := range testCases {
		got, err := decodeList(bytes.NewReader([]byte(tc.input[1:]))) // skip 'l'
		if err != nil {
			t.Errorf("parseList(%q) returned error: %v", tc.input, err)
			continue
		}

		if !reflect.DeepEqual(got, tc.expected) {
			t.Errorf("parseList(%q)\n=> got:  %#v\n=> want: %#v", tc.input, got, tc.expected)
		}
	}
}
