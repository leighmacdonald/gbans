package asset

import (
	"errors"
	"io"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/leighmacdonald/gbans/internal/domain"
)

func GenerateFileMeta(body io.Reader, name string) (string, string, int64, error) {
	content, errRead := io.ReadAll(body)
	if errRead != nil {
		return "", "", 0, errors.Join(errRead, domain.ErrReadContent)
	}

	mime := mimetype.Detect(content)

	if !strings.HasSuffix(strings.ToLower(name), mime.Extension()) {
		name += mime.Extension()
	}

	return name, mime.String(), int64(len(content)), nil
}
