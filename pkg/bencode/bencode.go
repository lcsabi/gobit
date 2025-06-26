package bencode

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// Value represents any valid bencode value. It may be one of:
//   - ByteString (string)
//   - Integer (int64)
//   - List ([]Value)
//   - Dictionary (map[string]Value)
//
// Reference: https://wiki.theory.org/BitTorrentSpecification#Bencoding
type Value any

// ByteString represents a bencoded byte string,
// which is always UTF-8 decoded and exposed as a Go string.
type ByteString = string

// Integer represents a bencoded integer.
type Integer = int64

// List represents a bencoded list of values.
type List = []Value

// Dictionary represents a bencoded dictionary with string keys and bencoded values.
type Dictionary = map[string]Value

// Decode reads bencoded data from the provided io.Reader and returns the corresponding
// Go representation as a Value. The result will be one of:
//   - ByteString (string)
//   - Integer (int64)
//   - List ([]Value)
//   - Dictionary (map[string]Value)
//
// This method reads the entire input into memory using io.ReadAll, making it suitable
// for .torrent files or other small bencode payloads. For large or streamed inputs,
// consider implementing a streaming Decoder.
//
// Returns an error if the input is invalid or incomplete.
func Decode(r io.Reader) (Value, error) {
	// TODO: optimize decoding for large torrent files and magnet links by introducing a Decoder type
	data, err := io.ReadAll(r) // ! possible bottleneck
	if err != nil {
		return nil, err
	}

	return parseBencode(bytes.NewReader(data))
}

// Encode encodes the given Value into its bencoded byte representation.
// Supported value types include:
//   - string or []byte → encoded as byte strings
//   - int or int64     → encoded as integers
//   - []Value   → encoded as a list
//   - map[string]Value → encoded as a dictionary with sorted keys
//
// The encoded data is returned as a new byte slice.
func Encode(val Value) ([]byte, error) {
	var buf bytes.Buffer
	err := EncodeTo(&buf, val)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// EncodeTo encodes the given Value and writes the result into the provided bytes.Buffer.
// This variant is more efficient for repeated encodings as it avoids reallocations.
//
// Returns an error if the input type is unsupported.
//
// Reference: https://wiki.theory.org/BitTorrentSpecification#Bencoding
func EncodeTo(w *bytes.Buffer, rawInput Value) error {
	switch input := rawInput.(type) {
	case []byte:
		return encodeByteString(w, string(input))

	case string:
		return encodeByteString(w, input)

	case int:
		return encodeInteger(w, int64(input))

	case int64:
		return encodeInteger(w, input)

	case []Value:
		return encodeList(w, input)

	case map[string]Value:
		return encodeDictionary(w, input)

	default:
		return fmt.Errorf("unsupported type %T", input)
	}
}

// TypeOf returns a short string description of the Value's type.
// Possible return values are: "byte string", "integer", "list", "dictionary", or "unknown".
func TypeOf(value Value) string {
	switch value.(type) {
	case ByteString:
		return "byte string"

	case Integer:
		return "integer"

	case List:
		return "list"

	case Dictionary:
		return "dictionary"

	default:
		return "unknown"
	}
}

// ToString returns a human-readable string representation of the given Value,
// formatted with indentation and type labels. This is useful for debugging.
func ToString(value Value) string {
	var sb strings.Builder
	prettyPrintValue(&sb, value, 0)

	return sb.String()
}

// TODO: test
// write error is not checked because we are always writing to a strings.Builder
func prettyPrintValue(w io.Writer, value Value, indentLevel int) {
	indent := strings.Repeat("  ", indentLevel)

	switch v := value.(type) {
	case ByteString:
		fmt.Fprintf(w, "%sstring: %q\n", indent, v)

	case Integer:
		fmt.Fprintf(w, "%sinteger: %d\n", indent, v)

	case List:
		fmt.Fprintf(w, "%slist:\n", indent)
		for i, item := range v {
			fmt.Fprintf(w, "%s  [%d]:\n", indent, i)
			prettyPrintValue(w, item, indentLevel+2)
		}

	case Dictionary:
		fmt.Fprintf(w, "%sdictionary:\n", indent)
		for k, val := range v {
			fmt.Fprintf(w, "%s  key: %q\n", indent, k)
			prettyPrintValue(w, val, indentLevel+2)
		}

	default:
		fmt.Fprintf(w, "%sunknown type: %T (%v)\n", indent, v, v)
	}
}

func parseBencode(r *bytes.Reader) (Value, error) {
	delimiter, err := r.ReadByte() // read beginning delimiter
	if err != nil {
		return nil, err
	}

	switch {
	case delimiter == 'i':
		return decodeInteger(r)

	case delimiter >= '0' && delimiter <= '9':
		return decodeByteString(r, delimiter) // delimiter is also the first digit of the byte string's length

	case delimiter == 'l':
		return decodeList(r)

	case delimiter == 'd':
		return decodeDictionary(r)

	default:
		return nil, fmt.Errorf("invalid bencode prefix: %c", delimiter)
	}
}

func decodeByteString(r *bytes.Reader, firstDigit byte) (ByteString, error) {
	// read the length of the byte string
	var buffer bytes.Buffer
	buffer.WriteByte(firstDigit)
	for {
		digit, err := r.ReadByte()
		if err != nil {
			return "", err
		}

		// delimiter for byte string length
		if digit == ':' {
			break
		}
		buffer.WriteByte(digit)
	}
	byteStringLength, err := strconv.ParseInt(buffer.String(), 10, 64)
	if err != nil {
		return "", err
	}

	// specify maximum length to prevent memory exhaustion
	const MaxByteStringLength = 10 * 1024 * 1024 // 10 MB
	if byteStringLength > MaxByteStringLength {
		return "", fmt.Errorf("byte string length too large: %d", byteStringLength)
	}

	byteString := make([]byte, byteStringLength) // read the byte string itself
	_, err = io.ReadFull(r, byteString)
	if err != nil {
		return "", err
	}

	return string(byteString), nil
}

func decodeInteger(r *bytes.Reader) (Integer, error) {
	var buffer bytes.Buffer
	first := true

	for {
		digit, err := r.ReadByte()
		if err != nil {
			return 0, err
		}

		if first {
			first = false
			nextDigit, err := r.ReadByte()
			if err != nil {
				return 0, fmt.Errorf("error peeking second digit: %w", err)
			}

			if digit == '-' && nextDigit == '0' {
				return 0, fmt.Errorf("negative zero in integer")
			}
			if digit == '0' && nextDigit != 'e' {
				return 0, fmt.Errorf("leading zero in integer")
			}

			// defensive unread, panic should not happen because we guarantee to read a byte before unreading
			if err := r.UnreadByte(); err != nil {
				return 0, fmt.Errorf("unread error while decoding integer: %w", err)
			}
		}

		if digit == 'e' {
			break
		}

		buffer.WriteByte(digit)
	}

	if buffer.Len() == 0 {
		return 0, errors.New("empty integer")
	}

	return strconv.ParseInt(buffer.String(), 10, 64)
}

func decodeList(r *bytes.Reader) (List, error) {
	var values List
	for {
		delimiter, err := r.ReadByte() // peek next type
		if err != nil {
			return nil, err
		}

		// end delimiter for lists
		if delimiter == 'e' {
			break
		}

		// defensive unread to properly identify next type
		// panic should not happen because we guarantee to read a byte before unreading
		if err := r.UnreadByte(); err != nil {
			return nil, fmt.Errorf("unread error while decoding integer: %w", err)
		}
		element, err := parseBencode(r)
		if err != nil {
			return nil, err
		}

		values = append(values, element)
	}

	return values, nil
}

func decodeDictionary(r *bytes.Reader) (Dictionary, error) {
	values := make(map[string]Value)
	for {
		delimiter, err := r.ReadByte() // peek next type
		if err != nil {
			return nil, err
		}
		// end delimiter for dictionaries
		if delimiter == 'e' {
			break
		}
		// defensive unread to properly identify next type
		// panic should not happen because we guarantee to read a byte before unreading
		if err := r.UnreadByte(); err != nil {
			return nil, fmt.Errorf("unread error while decoding integer: %w", err)
		}

		// parse the key
		key, err := parseBencode(r)
		if err != nil {
			return nil, err
		}

		// dictionaries must have byte strings as keys
		keyAsString, ok := key.(string)
		if !ok {
			return nil, errors.New("dictionary key is not a string")
		}

		// parse the value
		value, err := parseBencode(r)
		if err != nil {
			return nil, err
		}

		// append to hashmap
		values[keyAsString] = value
	}

	return values, nil
}

func encodeByteString(w *bytes.Buffer, value string) error {
	tmp := strconv.AppendInt(nil, int64(len(value)), 10) // append to a temporary byte slice
	w.Write(tmp)
	w.WriteByte(':')
	w.WriteString(value)

	return nil
}

func encodeInteger(w *bytes.Buffer, value int64) error {
	w.WriteByte('i')                                // beginning delimiter for an integer
	tmp := strconv.AppendInt(nil, int64(value), 10) // append to a temporary byte slice
	w.Write(tmp)
	w.WriteByte('e') // end delimiter for an integer

	return nil
}

func encodeList(w *bytes.Buffer, list []Value) error {
	w.WriteByte('l') // beginning delimiter for a list
	for _, item := range list {
		if err := EncodeTo(w, item); err != nil {
			return err
		}
	}
	w.WriteByte('e') // end delimiter for a list

	return nil
}

func encodeDictionary(w *bytes.Buffer, dictionary map[string]Value) error {
	w.WriteByte('d') // beginning delimiter for a dictionary
	keys := make([]string, 0, len(dictionary))
	for k := range dictionary {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if err := encodeByteString(w, k); err != nil {
			return err
		}
		if err := EncodeTo(w, dictionary[k]); err != nil {
			return err
		}
	}
	w.WriteByte('e') // end delimiter for a dictionary

	return nil
}
