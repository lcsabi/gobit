package util

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
)

// BencodedValue represents the possible values that can be parsed from a bencoded byte array.
//
// As per specification, it supports the following types: byte strings, integers, lists, and dictionaries.
type BencodedValue any

// Decode returns the parsed bencode that is read by the received io.Reader.
//
// Reference: https://wiki.theory.org/BitTorrentSpecification#Bencoding
func Decode(r io.Reader) (BencodedValue, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return parseBencode(bytes.NewReader(data))
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
		// delimiter is also the first digit of the Byte String's length
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
	for {
		digit, err := r.ReadByte()
		if err != nil {
			return 0, err
		}

		// end delimiter for integers
		if digit == 'e' {
			break
		}
		buffer.WriteByte(digit)
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

// TODO: create a bencode validator
