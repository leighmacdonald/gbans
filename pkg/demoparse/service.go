package demoparse

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"

	demostatsv1 "github.com/leighmacdonald/gbans/internal/demostats/v1"
	"github.com/leighmacdonald/gbans/internal/demostats/v1/demostatsv1connect"
)

var ErrDemoSubmit = errors.New("could not submit demo file")

func SubmitFile(ctx context.Context, url string, path string) (*Demo, error) {
	fileHandle, errDF := os.Open(path)
	if errDF != nil {
		return nil, errors.Join(errDF, ErrDemoSubmit)
	}
	defer fileHandle.Close()

	content, errRead := io.ReadAll(fileHandle)
	if errRead != nil {
		return nil, errors.Join(errRead, ErrDemoSubmit)
	}

	return Submit(ctx, url, fileHandle.Name(), bytes.NewReader(content))
}

// Submit uploads a raw .dem file to a tf2_demostats v0.3.x server over
// ConnectRPC (demostats.v1.DemoService/ParseDemo) and converts the typed
// response into a Demo.
//
// url is the base URL of the parser service, e.g. http://localhost:8811/.
func Submit(ctx context.Context, url string, name string, reader io.Reader) (*Demo, error) {
	content, errContent := io.ReadAll(reader)
	if errContent != nil && !errors.Is(errContent, io.ErrUnexpectedEOF) {
		return nil, errors.Join(errContent, ErrDemoSubmit)
	}

	client := demostatsv1connect.NewDemoServiceClient(http.DefaultClient, url)

	resp, errCall := client.ParseDemo(ctx, &demostatsv1.ParseDemoRequest{
		Demo:     content,
		Filename: name,
	})
	if errCall != nil {
		return nil, errors.Join(errCall, ErrDemoSubmit)
	}

	return DemoFromProto(resp, name), nil
}
