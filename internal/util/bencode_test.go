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
			t.Errorf("decodeInteger(%q) returned error: %v", tc.input, err)
			continue
		}

		if got != tc.expected {
			t.Errorf("decodeInteger(%q) => got: %v want: %d", tc.input, got, tc.expected)
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
			t.Errorf("decodeByteString(%q) returned error: %v", tc.input, err)
			continue
		}

		if got != tc.expected {
			t.Errorf("decodeByteString(%q) => got: %v want: %s", tc.input, got, tc.expected)
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
			t.Errorf("decodeList(%q) returned error: %v", tc.input, err)
			continue
		}

		if !reflect.DeepEqual(got, tc.expected) {
			t.Errorf("decodeList(%q) => got: %#v want: %#v", tc.input, got, tc.expected)
		}
	}
}

func TestParseDictionary(t *testing.T) {
	testCases := []struct {
		input    string
		expected map[string]BencodedValue
	}{
		{"d3:cow3:moo4:spam4:eggse", map[string]BencodedValue{"cow": "moo", "spam": "eggs"}},
		{"de", map[string]BencodedValue{}},
		{"d4:spaml1:a1:be3:inti3ee", map[string]BencodedValue{"spam": []BencodedValue{"a", "b"}, "int": int64(3)}},
	}

	for _, tc := range testCases {
		got, err := decodeDictionary(bytes.NewReader([]byte(tc.input[1:]))) // skip 'l'
		if err != nil {
			t.Errorf("decodeDictionary(%q) returned error: %v", tc.input, err)
			continue
		}

		if !reflect.DeepEqual(got, tc.expected) {
			t.Errorf("decodeDictionary(%q) => got: %#v want: %#v", tc.input, got, tc.expected)
		}
	}
}
