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
		got, err := decodeDictionary(bytes.NewReader([]byte(tc.input[1:]))) // skip 'd'
		if err != nil {
			t.Errorf("decodeDictionary(%q) returned error: %v", tc.input, err)
			continue
		}

		if !reflect.DeepEqual(got, tc.expected) {
			t.Errorf("decodeDictionary(%q) =>\ngot:\n%#v\nwant:\n%#v", tc.input, got, tc.expected)
		}
	}
}

func TestDecode(t *testing.T) {
	testCases := []struct {
		input    string
		expected map[string]BencodedValue
	}{
		{
			input: "d8:announce26:http://tracker.example.com10:created by13:ExampleClient4:infod6:lengthi123456e4:name13:test_file.txt12:piece lengthi262144e6:pieces20:aaaaaaaaaaaaaaaaaaaaee",
			expected: map[string]BencodedValue{
				"announce":   "http://tracker.example.com",
				"created by": "ExampleClient",
				"info": map[string]BencodedValue{
					"length":       int64(123456),
					"name":         "test_file.txt",
					"piece length": int64(262144),
					"pieces":       "aaaaaaaaaaaaaaaaaaaa",
				},
			},
		},
	}
	for _, tc := range testCases {
		got, err := Decode(bytes.NewReader([]byte(tc.input)))
		if err != nil {
			t.Errorf("Decode(%q) returned error: %v", tc.input, err)
			continue
		}

		if !reflect.DeepEqual(got, tc.expected) {
			t.Errorf("Decode(%q) =>\ngot:\n%#v\nwant:\n%#v", tc.input, got, tc.expected)
		}
	}
}

func TestEncodeByteString(t *testing.T) {
	tests := []struct {
		value    string
		expected string
	}{
		{"", "0:"},
		{"a", "1:a"},
		{"spam", "4:spam"},
		{"こんにちは", "15:こんにちは"}, // UTF-8: 5 runes, 15 bytes
	}

	for _, tt := range tests {
		var buf bytes.Buffer
		err := encodeByteString(&buf, tt.value)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if buf.String() != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, buf.String())
		}
	}
}

func TestEncodeInteger(t *testing.T) {
	tests := []struct {
		value    int64
		expected string
	}{
		{0, "i0e"},
		{42, "i42e"},
		{-7, "i-7e"},
		{123456789, "i123456789e"},
	}

	for _, tt := range tests {
		var buf bytes.Buffer
		err := encodeInteger(&buf, tt.value)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if buf.String() != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, buf.String())
		}
	}
}

func TestEncodeList(t *testing.T) {
	list := []BencodedValue{"spam", "eggs", 42}

	var buf bytes.Buffer
	err := encodeList(&buf, list)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "l4:spam4:eggsi42ee"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

func TestEncodeDictionary(t *testing.T) {
	dict := map[string]BencodedValue{
		"cow":   "moo",
		"spam":  "eggs",
		"count": 42,
	}

	var buf bytes.Buffer
	err := encodeDictionary(&buf, dict)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "d5:counti42e3:cow3:moo4:spam4:eggse"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

func TestEncode(t *testing.T) {
	input := map[string]BencodedValue{
		"info": map[string]BencodedValue{
			"name": "file.txt",
			"size": 1234,
		},
	}

	expected := "d4:infod4:name8:file.txt4:sizei1234eee"

	var buf bytes.Buffer
	res, err := Encode(input)
	if err != nil {
		t.Fatal(err)
	}
	if string(res) != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}
