// https://wiki.theory.org/BitTorrentSpecification#Bencoding

package util

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
)

type BencodedValue any

func Decode(r io.Reader) (BencodedValue, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return parseBencode(bytes.NewReader(data))
}

func parseBencode(r *bytes.Reader) (BencodedValue, error) {
	// read delimiter
	delimiter, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	switch {
	case delimiter == 'i':
		return parseInt(r)
	case delimiter >= '0' && delimiter <= '9':
		// delimiter is also the first digit of the Byte String's length
		return parseByteString(r, delimiter)
	default:
		return nil, fmt.Errorf("invalid bencode prefix: %c", delimiter)
	}
}

func parseInt(r *bytes.Reader) (int64, error) {
	var buffer bytes.Buffer
	for {
		digit, err := r.ReadByte()
		if err != nil {
			return 0, err
		}

		if digit == 'e' {
			break
		}
		buffer.WriteByte(digit)
	}

	return strconv.ParseInt(buffer.String(), 10, 64)
}

func parseByteString(r *bytes.Reader, firstDigit byte) (string, error) {
	var buffer bytes.Buffer
	buffer.WriteByte(firstDigit)

	for {
		digit, err := r.ReadByte()
		if err != nil {
			return "", err
		}

		if digit == ':' {
			break
		}
		buffer.WriteByte(digit)
	}
	byteStringLength, err := strconv.ParseInt(buffer.String(), 10, 64)
	if err != nil {
		return "", err
	}

	byteString := make([]byte, byteStringLength)
	_, err = io.ReadFull(r, byteString)
	if err != nil {
		return "", err
	}

	return string(byteString), nil
}
