package json

import (
	"encoding/json"
	"errors"
	"io"
)

var ErrDecodeJSON = errors.New("failed to decode JSON")

// Decode is a generic version of the stdlib json decoder.
func Decode[T any](reader io.Reader) (T, error) {
	var value T
	if err := json.NewDecoder(reader).Decode(&value); err != nil {
		return value, errors.Join(err, ErrDecodeJSON)
	}

	return value, nil
}
