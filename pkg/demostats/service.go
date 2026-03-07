package demostats

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/leighmacdonald/gbans/internal/json"
)

const defaultURL = "http://localhost:8811/"

var ErrDemoSubmit = errors.New("could not submit demo file")

func NewDefault() Client {
	return New(defaultURL, time.Second*60)
}

func New(url string, timeout time.Duration) Client {
	return Client{url: url, client: &http.Client{Timeout: timeout}}
}

// Client handles submitting demo files to a tf2_demostats backend service.
type Client struct {
	url    string
	client *http.Client
}

func (s Client) SubmitFile(ctx context.Context, path string) (*Demo, error) {
	fileHandle, errDF := os.Open(path)
	if errDF != nil {
		return nil, errors.Join(errDF, ErrDemoSubmit)
	}
	defer fileHandle.Close()

	content, errRead := io.ReadAll(fileHandle)
	if errRead != nil {
		return nil, errors.Join(errRead, ErrDemoSubmit)
	}

	return s.Submit(ctx, fileHandle.Name(), bytes.NewReader(content))
}

// Submit the contents of the demo reader via multipart form to the tf2_demostats backend service.
//
// Field for the demo: file
func (s Client) Submit(ctx context.Context, name string, reader io.Reader) (*Demo, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	part, errCreate := writer.CreateFormFile("file", name)
	if errCreate != nil {
		return nil, errors.Join(errCreate, ErrDemoSubmit)
	}

	content, errContent := io.ReadAll(reader)
	if errContent != nil && !errors.Is(errContent, io.ErrUnexpectedEOF) {
		return nil, errors.Join(errContent, ErrDemoSubmit)
	}

	if _, err := part.Write(content); err != nil {
		return nil, errors.Join(err, ErrDemoSubmit)
	}

	if errClose := writer.Close(); errClose != nil {
		return nil, errors.Join(errClose, ErrDemoSubmit)
	}

	req, errReq := http.NewRequestWithContext(ctx, http.MethodPost, s.url, body)
	if errReq != nil {
		return nil, errors.Join(errReq, ErrDemoSubmit)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, errSend := s.client.Do(req)
	if errSend != nil {
		return nil, errors.Join(errSend, ErrDemoSubmit)
	}
	defer resp.Body.Close()

	demo, errDecode := json.Decode[Demo](resp.Body)
	if errDecode != nil {
		return nil, errors.Join(errDecode, ErrDemoSubmit)
	}

	return &demo, nil
}
