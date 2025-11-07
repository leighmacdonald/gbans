package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-querystring/query"
	gjson "github.com/leighmacdonald/gbans/internal/json"
	"github.com/stretchr/testify/require"
)

func GetNotFound(t *testing.T, router http.Handler, path string) {
	t.Helper()

	endpoint(t, router, http.MethodGet, path, nil, http.StatusNotFound, nil)
}

func GetGOK[T any](t *testing.T, router http.Handler, path string, body ...any) T { //nolint:ireturn
	t.Helper()
	if len(body) > 0 {
		return endpointWithReceiver[T](t, router, http.MethodGet, path, body[0], http.StatusOK, nil)
	}

	return endpointWithReceiver[T](t, router, http.MethodGet, path, nil, http.StatusOK, nil)
}

func GetOKBytes(t *testing.T, router http.Handler, path string) []byte {
	t.Helper()
	response := endpoint(t, router, http.MethodGet, path, nil, http.StatusOK, nil)

	return response.Body.Bytes()
}

func GetForbidden(t *testing.T, router http.Handler, path string) {
	t.Helper()
	endpoint(t, router, http.MethodGet, path, nil, http.StatusForbidden, nil)
}

func PutForbidden(t *testing.T, router http.Handler, path string, body any) {
	t.Helper()
	endpoint(t, router, http.MethodPut, path, body, http.StatusForbidden, nil)
}

func PutGOK[T any](t *testing.T, router http.Handler, path string, body any) T { //nolint:ireturn
	t.Helper()

	return endpointWithReceiver[T](t, router, http.MethodPut, path, body, http.StatusOK, nil)
}

func PutOK(t *testing.T, router http.Handler, path string, body any) {
	t.Helper()
	endpoint(t, router, http.MethodPut, path, body, http.StatusOK, nil)
}

func PostAccepted(t *testing.T, router http.Handler, path string, body any) {
	t.Helper()
	endpoint(t, router, http.MethodPost, path, body, http.StatusAccepted, nil)
}

func PostConflict(t *testing.T, router http.Handler, path string, body any) {
	t.Helper()
	endpoint(t, router, http.MethodPost, path, body, http.StatusConflict, nil)
}

func PostCreatedForm[T any](t *testing.T, router http.Handler, path string, body any, headers map[string]string) T { //nolint:ireturn
	t.Helper()

	return endpointWithReceiver[T](t, router, http.MethodPost, path, body, http.StatusCreated, headers)
}

func PostGCreated[T any](t *testing.T, router http.Handler, path string, body any) T { //nolint:ireturn
	t.Helper()

	return endpointWithReceiver[T](t, router, http.MethodPost, path, body, http.StatusCreated, nil)
}

func PostGOK[T any](t *testing.T, router http.Handler, path string, body any) T { //nolint:ireturn
	t.Helper()

	return endpointWithReceiver[T](t, router, http.MethodPost, path, body, http.StatusOK, nil)
}

func PostOK(t *testing.T, router http.Handler, path string, body any) {
	t.Helper()
	endpoint(t, router, http.MethodPost, path, body, http.StatusOK, nil)
}

func DeleteOK(t *testing.T, router http.Handler, path string, body any) {
	t.Helper()
	endpoint(t, router, http.MethodDelete, path, body, http.StatusOK, nil)
}

func endpointWithReceiver[T any](t *testing.T, router http.Handler, method string, //nolint:ireturn
	path string, body any, expectedStatus int, headers map[string]string,
) T {
	t.Helper()

	resp := endpoint(t, router, method, path, body, expectedStatus, headers)
	value, err := gjson.Decode[T](resp.Body)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	return value
}

func endpoint(t *testing.T, router http.Handler, method string, path string, body any, expectedStatus int, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	reqCtx, cancel := context.WithTimeout(t.Context(), time.Second*10)
	defer cancel()

	recorder := httptest.NewRecorder()

	var bodyReader io.Reader
	if body != nil {
		bodyJSON, errJSON := json.Marshal(body)
		if errJSON != nil {
			t.Fatalf("Failed to encode request: %v", errJSON)
		}

		bodyReader = bytes.NewReader(bodyJSON)
	}

	if body != nil && method == http.MethodGet {
		values, err := query.Values(body)
		if err != nil {
			t.Fatalf("failed to encode values: %v", err)
		}

		path += "?" + values.Encode()
	}

	request, errRequest := http.NewRequestWithContext(reqCtx, method, path, bodyReader)
	if errRequest != nil {
		t.Fatalf("Failed to make request: %v", errRequest)
	}

	for key, value := range headers {
		request.Header.Set(key, value)
	}

	router.ServeHTTP(recorder, request)

	require.Equal(t, expectedStatus, recorder.Code, "Received invalid response code. method: %s path: %s", method, path)

	return recorder
}
