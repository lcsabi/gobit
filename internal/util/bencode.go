package util

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
)

// BencodedValue represents the possible values that can be parsed from a bencoded byte array.
// As per specification, it supports the following types: byte strings, integers, lists, and dictionaries.
type BencodedValue any

// Decode reads bencoded data from the provided io.Reader and returns the
// corresponding Go representation as a BencodedValue.
//
// The returned BencodedValue is one of the following supported Go types:
//   - string                    → for bencoded byte strings
//   - int64                     → for bencoded integers
//   - []BencodedValue           → for bencoded lists
//   - map[string]BencodedValue  → for bencoded dictionaries
//
// Internally, Decode reads the entire input into memory using io.ReadAll,
// which is suitable for typical .torrent files under 1MB. For large inputs
// or streamed magnet links, consider implementing a streaming parser.
//
// Returns an error if the input is invalid or unreadable.
//
// Reference: https://wiki.theory.org/BitTorrentSpecification#Bencoding
func Decode(r io.Reader) (BencodedValue, error) {
	// TODO: optimize decoding for large torrent files and magnet links
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
//   - string or []byte → encoded as byte strings
//   - int or int64     → encoded as integers
//   - []BencodedValue  → encoded as a list
//   - map[string]BencodedValue → encoded as a dictionary (keys are sorted lexicographically)
//
// Reference: https://wiki.theory.org/BitTorrentSpecification#Bencoding
func Encode(val BencodedValue) ([]byte, error) {
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
func EncodeTo(w *bytes.Buffer, rawInput BencodedValue) error {
	switch input := rawInput.(type) {
	case []byte:
		return encodeByteString(w, string(input))
	case string:
		return encodeByteString(w, input)
	case int:
		return encodeInteger(w, int64(input))
	case int64:
		return encodeInteger(w, input)
	case []BencodedValue:
		return encodeList(w, input)
	case map[string]BencodedValue:
		return encodeDictionary(w, input)
	default:
		return fmt.Errorf("unsupported type %T", input)
	}
}

func parseBencode(r *bytes.Reader) (BencodedValue, error) {
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

func decodeByteString(r *bytes.Reader, firstDigit byte) (string, error) {
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

func decodeInteger(r *bytes.Reader) (int64, error) {
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

func decodeList(r *bytes.Reader) ([]BencodedValue, error) {
	var values []BencodedValue
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

func decodeDictionary(r *bytes.Reader) (map[string]BencodedValue, error) {
	values := make(map[string]BencodedValue)
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

func encodeList(w *bytes.Buffer, list []BencodedValue) error {
	w.WriteByte('l') // beginning delimiter for a list
	for _, item := range list {
		if err := EncodeTo(w, item); err != nil {
			return err
		}
	}
	w.WriteByte('e') // end delimiter for a list

	return nil
}

func encodeDictionary(w *bytes.Buffer, dictionary map[string]BencodedValue) error {
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
