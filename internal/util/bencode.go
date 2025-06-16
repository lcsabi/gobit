package util

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
)

// BencodedValue represents the possible values that can be parsed from a bencoded byte array.
//
// As per specification, it supports the following types: byte strings, integers, lists, and dictionaries.
//
// Reference: https://wiki.theory.org/BitTorrentSpecification#Bencoding
type BencodedValue any

// Decode returns the parsed bencode that is read by the received io.Reader.
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
	default:
		return nil, fmt.Errorf("invalid bencode prefix: %c", delimiter)
	}
}

func decodeByteString(r *bytes.Reader, firstDigit byte) (string, error) {
	var buffer bytes.Buffer
	buffer.WriteByte(firstDigit)

	// read the length of the byte string
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
		delimiter, err := r.ReadByte()
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
