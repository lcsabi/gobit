package util

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// BencodeValue represents every possible value that can be parsed from a bencoded byte array.
//
// As per specification, it supports the following types: byte strings, integers, lists, and dictionaries.
//
// Reference: https://wiki.theory.org/BitTorrentSpecification#Bencoding
type BencodeValue any

// BencodeByteString represents a bencoded byte string (always UTF-8 decoded).
type BencodeByteString = string

// BencodeInteger represents a bencoded integer.
type BencodeInteger = int64

// BencodeList represents a bencoded list of values.
type BencodeList = []BencodeValue

// BencodeDictionary represents a bencoded dictionary with string keys and bencoded values.
type BencodeDictionary = map[string]BencodeValue

// Decode reads bencoded data from the provided io.Reader and returns the
// corresponding Go representation as a BencodedValue.
//
// The returned BencodedValue is one of the following:
//   - BencodeByteString (string)
//   - BencodeInteger (int64)
//   - BencodeList ([]BencodedValue)
//   - BencodeDictionary (map[string]BencodedValue)
//
// Internally, Decode reads the entire input into memory using io.ReadAll,
// which is suitable for typical .torrent files under 1MB. For large inputs
// or streamed magnet links, consider implementing a streaming parser.
//
// Returns an error if the input is invalid or unreadable.
//
// Reference: https://wiki.theory.org/BitTorrentSpecification#Bencoding
func Decode(r io.Reader) (BencodeValue, error) {
	// TODO: optimize decoding for large torrent files and magnet links by introducing a Decoder type
	data, err := io.ReadAll(r) // ! possible bottleneck
	if err != nil {
		return nil, err
	}

	return parseBencode(bytes.NewReader(data))
}

// Encode returns the bencoded byte representation of the given BencodedValue.
//
// It allocates and uses an internal bytes.Buffer, and is suitable for
// general-purpose use cases where you want the encoded output as a []byte.
//
// The input must be a valid BencodedValue. Otherwise, an error is returned.
//
// Supported types are:
//   - string or []byte         -> encoded as byte strings
//   - int or int64             -> encoded as integers
//   - []BencodedValue          -> encoded as a list
//   - map[string]BencodedValue -> encoded as a dictionary (keys are sorted lexicographically)
//
// Reference: https://wiki.theory.org/BitTorrentSpecification#Bencoding
func Encode(val BencodeValue) ([]byte, error) {
	var buf bytes.Buffer
	err := EncodeTo(&buf, val)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// EncodeTo writes the bencoded representation of the given BencodedValue
// directly into the provided bytes.Buffer.
//
// This function is useful for high-performance encoding where you want to
// avoid unnecessary allocations by reusing a buffer.
//
// Returns an error if the input type is unsupported.
//
// Reference: https://wiki.theory.org/BitTorrentSpecification#Bencoding
func EncodeTo(w *bytes.Buffer, rawInput BencodeValue) error {
	switch input := rawInput.(type) {
	case []byte:
		return encodeByteString(w, string(input))

	case string:
		return encodeByteString(w, input)

	case int:
		return encodeInteger(w, int64(input))

	case int64:
		return encodeInteger(w, input)

	case []BencodeValue:
		return encodeList(w, input)

	case map[string]BencodeValue:
		return encodeDictionary(w, input)

	default:
		return fmt.Errorf("unsupported type %T", input)
	}
}

func TypeOf(value BencodeValue) string {
	switch value.(type) {
	case BencodeByteString:
		return "byte string"

	case BencodeInteger:
		return "integer"

	case BencodeList:
		return "list"

	case BencodeDictionary:
		return "dictionary"

	default:
		return "unknown"
	}
}

func ToString(value BencodeValue) string {
	var sb strings.Builder
	PrettyPrintBencodeValue(&sb, value, 0)

	return sb.String()
}

func PrettyPrintBencodeValue(w io.Writer, value BencodeValue, indentLevel int) {
	indent := strings.Repeat("  ", indentLevel)

	switch v := value.(type) {
	case BencodeByteString:
		fmt.Fprintf(w, "%sstring: %q\n", indent, v)

	case BencodeInteger:
		fmt.Fprintf(w, "%sinteger: %d\n", indent, v)

	case BencodeList:
		fmt.Fprintf(w, "%slist:\n", indent)
		for i, item := range v {
			fmt.Fprintf(w, "%s  [%d]:\n", indent, i)
			PrettyPrintBencodeValue(w, item, indentLevel+2)
		}

	case BencodeDictionary:
		fmt.Fprintf(w, "%sdictionary:\n", indent)
		for k, val := range v {
			fmt.Fprintf(w, "%s  key: %q\n", indent, k)
			PrettyPrintBencodeValue(w, val, indentLevel+2)
		}

	default:
		fmt.Fprintf(w, "%sunknown type: %T (%v)\n", indent, v, v)
	}
}

func parseBencode(r *bytes.Reader) (BencodeValue, error) {
	// read beginning delimiter
	delimiter, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	switch {
	case delimiter == 'i':
		return decodeInteger(r)

	case delimiter >= '0' && delimiter <= '9':
		// delimiter is also the first digit of the byte string's length
		return decodeByteString(r, delimiter)

	case delimiter == 'l':
		return decodeList(r)

	case delimiter == 'd':
		return decodeDictionary(r)

	default:
		return nil, fmt.Errorf("invalid bencode prefix: %c", delimiter)
	}
}

func decodeByteString(r *bytes.Reader, firstDigit byte) (BencodeByteString, error) {
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

	// read the byte string itself
	byteString := make([]byte, byteStringLength)
	_, err = io.ReadFull(r, byteString)
	if err != nil {
		return "", err
	}

	return string(byteString), nil
}

func decodeInteger(r *bytes.Reader) (BencodeInteger, error) {
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

			// defensive unread, panic should not happen because
			// we guarantee to read a byte before unreading
			if err := r.UnreadByte(); err != nil {
				return 0, fmt.Errorf("unread error: %w", err)
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

func decodeList(r *bytes.Reader) (BencodeList, error) {
	var values []BencodeValue
	for {
		delimiter, err := r.ReadByte() // peek next type
		if err != nil {
			return nil, err
		}

		// end delimiter for lists
		if delimiter == 'e' {
			break
		}

		r.UnreadByte() // unread to properly identify next type
		element, err := parseBencode(r)
		if err != nil {
			return nil, err
		}

		values = append(values, element)
	}

	return values, nil
}

func decodeDictionary(r *bytes.Reader) (BencodeDictionary, error) {
	values := make(map[string]BencodeValue)
	for {
		delimiter, err := r.ReadByte() // peek next type
		if err != nil {
			return nil, err
		}
		// end delimiter for dictionaries
		if delimiter == 'e' {
			break
		}
		r.UnreadByte() // unread to properly identify next type

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

func encodeList(w *bytes.Buffer, list []BencodeValue) error {
	w.WriteByte('l') // beginning delimiter for a list
	for _, item := range list {
		if err := EncodeTo(w, item); err != nil {
			return err
		}
	}
	w.WriteByte('e') // end delimiter for a list

	return nil
}

func encodeDictionary(w *bytes.Buffer, dictionary map[string]BencodeValue) error {
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

// TODO: add a String() method to pretty-print BencodedValue
