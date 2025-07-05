package bencode

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// TestDecode verifies recursive decoding of a complete bencoded structure,
// such as a torrent metadata dictionary.
func TestDecode(t *testing.T) {
	testCases := []struct {
		input    string
		expected Dictionary
	}{
		{
			input: "d8:announce26:http://tracker.example.com10:created by13:ExampleClient4:infod6:lengthi123456e4:name13:test_file.txt12:piece lengthi262144e6:pieces20:aaaaaaaaaaaaaaaaaaaaee",
			expected: Dictionary{
				"announce":   "http://tracker.example.com",
				"created by": "ExampleClient",
				"info": Dictionary{
					"length":       int64(123456),
					"name":         "test_file.txt",
					"piece length": int64(262144),
					"pieces":       "aaaaaaaaaaaaaaaaaaaa",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := Decode(bytes.NewReader([]byte(tc.input)))
			if err != nil {
				t.Errorf("Decode(%q) returned error: %v", tc.input, err)
				return
			}

			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("Decode(%q) =>\ngot:\n%#v\nwant:\n%#v", tc.input, got, tc.expected)
			}
		})
	}
}

// TestEncode performs end-to-end encoding of a complex, nested bencoded dictionary.
func TestEncode(t *testing.T) {
	input := Dictionary{
		"announce":   "http://tracker.example.com",
		"created by": "ExampleClient",
		"info": map[string]Value{
			"length":       int64(123456),
			"name":         "test_file.txt",
			"piece length": int64(262144),
			"pieces":       "aaaaaaaaaaaaaaaaaaaa",
		},
	}

	expected := "d8:announce26:http://tracker.example.com10:created by13:ExampleClient4:infod6:lengthi123456e4:name13:test_file.txt12:piece lengthi262144e6:pieces20:aaaaaaaaaaaaaaaaaaaaee"

	res, err := Encode(input)
	if err != nil {
		t.Fatal(err)
	}
	if string(res) != expected {
		t.Errorf("expected %q, got %q", expected, string(res))
	}
}

// TestTypeOf checks the behavior of TypeOf for each valid and invalid bencode Value type.
// It ensures that the returned string matches the expected classification.
func TestTypeOf(t *testing.T) {
	tests := []struct {
		name     string
		input    Value
		expected string
	}{
		{"String", ByteString("hello"), "byte string"},
		{"Integer", Integer(42), "integer"},
		{"List", List{ByteString("spam")}, "list"},
		{"Dictionary", Dictionary{}, "dictionary"},
		{"Unknown", struct{}{}, "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := TypeOf(tc.input)
			if !strings.HasPrefix(got, tc.expected) {
				t.Errorf("TypeOf(%v) = %q; want prefix %q", tc.input, got, tc.expected)
			}
		})
	}
}

// TODO: test ToString here

// TestAsByteString verifies correct type assertion behavior of AsByteString.
func TestAsByteString(t *testing.T) {
	tests := []struct {
		name    string
		input   Value
		expects ByteString
		wantErr bool
	}{
		{
			"valid byte string",
			ByteString("hello"),
			ByteString("hello"),
			false,
		},
		{
			"invalid byte string",
			Integer(99),
			"",
			true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := AsByteString(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				expected := fmt.Sprintf("expected ByteString, got %T", tc.input)
				if err.Error() != expected {
					t.Errorf("expected error %q, got %q", expected, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expects {
				t.Errorf("expected %q, got %q", tc.expects, got)
			}
		})
	}
}

// TestAsInteger verifies correct type assertion behavior of AsInteger.
func TestAsInteger(t *testing.T) {
	tests := []struct {
		name    string
		input   Value
		expects Integer
		wantErr bool
	}{
		{"valid integer", Integer(42), 42, false},
		{"invalid integer", ByteString("x"), 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := AsInteger(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				expected := fmt.Sprintf("expected Integer, got %T", tc.input)
				if err.Error() != expected {
					t.Errorf("expected error %q, got %q", expected, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expects {
				t.Errorf("expected %v, got %v", tc.expects, got)
			}
		})
	}
}

// TestAsList verifies correct type assertion behavior of AsList.
func TestAsList(t *testing.T) {
	tests := []struct {
		name    string
		input   Value
		expects List
		wantErr bool
	}{
		{"valid list", List{Integer(1), Integer(2)}, List{Integer(1), Integer(2)}, false},
		{"invalid list", "not a list", nil, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := AsList(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				expected := fmt.Sprintf("expected List, got %T", tc.input)
				if err.Error() != expected {
					t.Errorf("expected error %q, got %q", expected, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.expects) {
				t.Errorf("expected list of length %d, got %d", len(tc.expects), len(got))
			}
		})
	}
}

// TestAsDictionary verifies correct type assertion behavior of AsDictionary.
func TestAsDictionary(t *testing.T) {
	tests := []struct {
		name    string
		input   Value
		expects Dictionary
		wantErr bool
	}{
		{
			"valid dictionary",
			Dictionary{"key": Integer(1)},
			Dictionary{"key": Integer(1)},
			false,
		},
		{
			"invalid dictionary",
			Integer(42),
			nil,
			true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := AsDictionary(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				expected := fmt.Sprintf("expected Dictionary, got %T", tc.input)
				if err.Error() != expected {
					t.Errorf("expected error %q, got %q", expected, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.expects) {
				t.Errorf("expected dictionary of length %d, got %d", len(tc.expects), len(got))
			}
		})
	}
}

// TestConvertListToByteStrings checks correct conversion of a bencoded list to []ByteString.
func TestConvertListToByteStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    List
		expected []ByteString
		wantErr  bool
		errSub   string
	}{
		{
			"valid ByteString list",
			List{ByteString("file"), ByteString("name"), ByteString("txt")},
			[]ByteString{"file", "name", "txt"},
			false,
			"",
		},
		{
			"list with non-ByteString element",
			List{ByteString("valid"), 123, ByteString("another")},
			nil,
			true,
			"element at index 1",
		},
		{
			"empty list",
			List{},
			[]ByteString{},
			false,
			"",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ConvertListToByteStrings(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.errSub) {
					t.Errorf("expected error to contain %q, got %v", tc.errSub, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.expected) {
				t.Fatalf("expected length %d, got %d", len(tc.expected), len(got))
			}
			for i := range got {
				if got[i] != tc.expected[i] {
					t.Errorf("element %d mismatch: expected %q, got %q", i, tc.expected[i], got[i])
				}
			}
		})
	}
}

// TestConvertListToIntegers checks correct conversion of a bencoded list to []Integer.
func TestConvertListToIntegers(t *testing.T) {
	tests := []struct {
		name     string
		input    List
		expected []Integer
		wantErr  bool
		errSub   string
	}{
		{
			"valid Integer list",
			List{Integer(1), Integer(2), Integer(3)},
			[]Integer{1, 2, 3},
			false,
			"",
		},
		{
			"list with non-Integer element",
			List{Integer(1), Integer(2), ByteString("three")},
			nil,
			true,
			"element at index 2",
		},
		{
			"empty list",
			List{},
			[]Integer{},
			false,
			"",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ConvertListToIntegers(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.errSub) {
					t.Errorf("expected error to contain %q, got %v", tc.errSub, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.expected) {
				t.Fatalf("expected length %d, got %d", len(tc.expected), len(got))
			}
			for i := range got {
				if got[i] != tc.expected[i] {
					t.Errorf("element %d mismatch: expected %d, got %d", i, tc.expected[i], got[i])
				}
			}
		})
	}
}

// TODO: test prettyPrintValue here

// TestParseString verifies decoding of bencoded strings.
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
		t.Run(tc.input, func(t *testing.T) {
			got, err := decodeByteString(bytes.NewReader([]byte(tc.input[1:])), tc.input[0]) // skip first digit
			if err != nil {
				t.Errorf("decodeByteString(%q) returned error: %v", tc.input, err)
				return
			}

			if got != tc.expected {
				t.Errorf("decodeByteString(%q) => got: %v want: %s", tc.input, got, tc.expected)
			}
		})
	}
}

// TestDecodeInvalidByteString ensures that malformed byte strings return an error.
func TestDecodeInvalidByteString(t *testing.T) {
	testCases := []string{
		"4spam", // missing colon
		"999:",  // declared length longer than actual
		"3:ab",  // declared length shorter than actual
		"a:b",   // non-numeric length
	}

	for _, input := range testCases {
		t.Run(input, func(t *testing.T) {
			_, err := decodeByteString(bytes.NewReader([]byte(input[1:])), input[0])
			if err == nil {
				t.Errorf("expected error for input %q, got nil", input)
			}
		})
	}
}

// TestParseInteger verifies decoding of bencoded integers.
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

// TestDecodeInvalidInteger ensures that malformed integers return an error.
func TestDecodeInvalidInteger(t *testing.T) {
	testCases := []string{
		"ie",     // empty integer
		"i-0e",   // negative zero
		"i123",   // missing 'e'
		"i12a3e", // invalid character in integer
		"i02e",   // leading zero
	}

	for _, input := range testCases {
		t.Run(input, func(t *testing.T) {
			_, err := decodeInteger(bytes.NewReader([]byte(input[1:]))) // skip 'i'
			if err == nil {
				t.Errorf("expected error for input %q, got nil", input)
			}
		})
	}
}

// TestParseList verifies decoding of bencoded lists containing strings and integers.
func TestParseList(t *testing.T) {
	testCases := []struct {
		input    string
		expected []Value
	}{
		{"l4:spam4:eggse", []Value{"spam", "eggs"}},
		{"le", nil},
		{"li1ei20e4:spame", []Value{int64(1), int64(20), "spam"}},
		{"l3:mooi42ee", []Value{"moo", int64(42)}},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := decodeList(bytes.NewReader([]byte(tc.input[1:]))) // skip 'l'
			if err != nil {
				t.Errorf("decodeList(%q) returned error: %v", tc.input, err)
				return
			}

			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("decodeList(%q) => got: %#v want: %#v", tc.input, got, tc.expected)
			}
		})
	}
}

// TestDecodeInvalidList ensures that malformed lists return an error.
func TestDecodeInvalidList(t *testing.T) {
	testCases := []string{
		"li1ei2e",      // missing ending 'e'
		"l4:spam4eggs", // malformed string
		"lxe",          // unknown type in list
	}

	for _, input := range testCases {
		t.Run(input, func(t *testing.T) {
			_, err := decodeList(bytes.NewReader([]byte(input[1:]))) // skip 'l'
			if err == nil {
				t.Errorf("expected error for input %q, got nil", input)
			}
		})
	}
}

// TestParseDictionary verifies decoding of bencoded dictionaries with mixed value types.
func TestParseDictionary(t *testing.T) {
	testCases := []struct {
		input    string
		expected Dictionary
	}{
		{"d3:cow3:moo4:spam4:eggse", Dictionary{"cow": "moo", "spam": "eggs"}},
		{"de", Dictionary{}},
		{"d4:spaml1:a1:be3:inti3ee", Dictionary{"spam": List{"a", "b"}, "int": int64(3)}},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := decodeDictionary(bytes.NewReader([]byte(tc.input[1:]))) // skip 'd'
			if err != nil {
				t.Errorf("decodeDictionary(%q) returned error: %v", tc.input, err)
				return
			}

			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("decodeDictionary(%q) =>\ngot:\n%#v\nwant:\n%#v", tc.input, got, tc.expected)
			}
		})
	}
}

// TestDecodeInvalidDictionary ensures that malformed dictionaries return an error.
func TestDecodeInvalidDictionary(t *testing.T) {
	testCases := []string{
		"d3:cowmoo4:spam4:eggse", // missing colon in "cow": "moo"
		"d3:cow3:moo3:spam",      // missing value for last key
		"d3:cowi42e3:mooe",       // bad key
	}

	for _, input := range testCases {
		t.Run(input, func(t *testing.T) {
			_, err := decodeDictionary(bytes.NewReader([]byte(input[1:]))) // skip 'd'
			if err == nil {
				t.Errorf("expected error for input %q, got nil", input)
			}
		})
	}
}

// TestDecodeUnknownType ensures that unrecognized bencode type characters return an error.
func TestDecodeUnknownType(t *testing.T) {
	input := "x12345e"
	_, err := Decode(bytes.NewReader([]byte(input)))
	if err == nil {
		t.Errorf("expected error for unknown type in input %q, got nil", input)
	}
}

// TestEncodeByteString checks encoding of various UTF-8 and ASCII strings into bencode format.
func TestEncodeByteString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "0:"},
		{"a", "1:a"},
		{"spam", "4:spam"},
		{"こんにちは", "15:こんにちは"}, // UTF-8: 5 runes, 15 bytes
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			var buf bytes.Buffer
			err := encodeByteString(&buf, tc.input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if buf.String() != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, buf.String())
			}
		})
	}
}

// TestEncodeInteger checks encoding of positive, negative, and large integers into bencode format.
func TestEncodeInteger(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{"0", 0, "i0e"},
		{"42", 42, "i42e"},
		{"-7", -7, "i-7e"},
		{"123456789", 123456789, "i123456789e"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := encodeInteger(&buf, tc.input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if buf.String() != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, buf.String())
			}
		})
	}
}

// TestEncodeList verifies encoding of lists containing strings and integers into bencode format.
func TestEncodeList(t *testing.T) {
	list := []Value{"spam", "eggs", 42}

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

// TestEncodeDictionary verifies encoding of dictionaries with string and integer values into bencode format.
func TestEncodeDictionary(t *testing.T) {
	dict := map[string]Value{
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

// TODO: implement benchmarking decode and encode
// TODO: test large payloads (10MB+)
// TODO: test maximum byte string length
