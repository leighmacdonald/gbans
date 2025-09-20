package json

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

var ErrDecodeJSON = errors.New("failed to decode JSON")

func Decode[T any](reader io.Reader) (T, error) {
	var value T
	if err := json.NewDecoder(reader).Decode(&value); err != nil {
		return value, errors.Join(err, ErrDecodeJSON)
	}

	return value, nil
}

func DecodeBytes[T any](body []byte) (T, error) {
	return Decode[T](bytes.NewReader(body))
}
