package demoparse

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
)

var ErrDemoSubmit = errors.New("could not submit demo file")

func Submit(ctx context.Context, url string, path string) (*Demo, error) {
	fileHandle, errDF := os.Open(path)
	if errDF != nil {
		return nil, errors.Join(errDF, ErrDemoSubmit)
	}

	content, errContent := io.ReadAll(fileHandle)
	if errContent != nil {
		return nil, errors.Join(errDF, ErrDemoSubmit)
	}

	info, errInfo := fileHandle.Stat()
	if errInfo != nil {
		return nil, errors.Join(errInfo, ErrDemoSubmit)
	}

	defer fileHandle.Close()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	part, errCreate := writer.CreateFormFile("file", info.Name())
	if errCreate != nil {
		return nil, errors.Join(errCreate, ErrDemoSubmit)
	}

	if _, err := part.Write(content); err != nil {
		return nil, errors.Join(errCreate, ErrDemoSubmit)
	}

	if errClose := writer.Close(); errClose != nil {
		return nil, errors.Join(errClose, ErrDemoSubmit)
	}

	req, errReq := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if errReq != nil {
		return nil, errors.Join(errReq, ErrDemoSubmit)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, errSend := client.Do(req)
	if errSend != nil {
		return nil, errors.Join(errSend, ErrDemoSubmit)
	}

	defer resp.Body.Close()

	var demo Demo

	// TODO remove this extra copy once this feature doesnt have much need for debugging/inspection.
	rawBody, errRead := io.ReadAll(resp.Body)
	if errRead != nil {
		return nil, errors.Join(errRead, ErrDemoSubmit)
	}

	if errDecode := json.NewDecoder(bytes.NewReader(rawBody)).Decode(&demo); errDecode != nil {
		return nil, errors.Join(errDecode, ErrDemoSubmit)
	}

	return &demo, nil
}
