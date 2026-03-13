package zstd

import (
	"errors"
	"io"

	"github.com/klauspost/compress/zstd"
)

const Extension = ".zstd"

var (
	ErrCompress = errors.New("failed to compress data")

	NewReader = zstd.NewReader //nolint:gochecknoglobals
)

func Compress(input io.Reader, output io.Writer) error {
	writer, errWriter := zstd.NewWriter(output)
	if errWriter != nil {
		return errors.Join(errWriter, ErrCompress)
	}

	// We do not use a defer for Close because it must be flushed.
	if _, errCopy := io.Copy(writer, input); errCopy != nil {
		if err := writer.Close(); err != nil {
			return errors.Join(err, ErrCompress)
		}

		return errors.Join(errCopy, errCopy)
	}

	if err := writer.Close(); err != nil {
		return errors.Join(err, ErrCompress)
	}

	return nil
}

func Decompress(input io.ReadCloser) ([]byte, error) {
	reader, errReader := zstd.NewReader(input)
	if errReader != nil {
		return nil, errors.Join(errReader, ErrCompress)
	}

	defer reader.Close()

	body, errBody := io.ReadAll(reader)
	if errBody != nil {
		return nil, errors.Join(errBody, ErrCompress)
	}

	return body, nil
}
