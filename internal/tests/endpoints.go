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
	"github.com/stretchr/testify/require"
)

func GetOKBytes(t *testing.T, router http.Handler, path string) []byte {
	t.Helper()

	response := endpoint(t, router, http.MethodGet, path, nil, http.StatusOK, nil)
	return response.Body.Bytes()
}

func GetForbidden(t *testing.T, router http.Handler, path string, receiver ...any) {
	t.Helper()

	if len(receiver) > 0 {
		endpointWithReceiver(t, router, http.MethodGet, path, nil, http.StatusForbidden, nil, receiver[0])
	} else {
		endpoint(t, router, http.MethodGet, path, nil, http.StatusForbidden, nil)
	}
}

func PutForbidden(t *testing.T, router http.Handler, path string, body any, receiver ...any) {
	t.Helper()

	if len(receiver) > 0 {
		endpointWithReceiver(t, router, http.MethodPut, path, body, http.StatusForbidden, nil, receiver[0])
	} else {
		endpoint(t, router, http.MethodPut, path, body, http.StatusForbidden, nil)
	}
}

func GetNotFound(t *testing.T, router http.Handler, path string, receiver ...any) {
	t.Helper()

	endpointWithReceiver(t, router, http.MethodGet, path, nil, http.StatusNotFound, nil, receiver)
}

func GetOK(t *testing.T, router http.Handler, path string, receiver ...any) {
	t.Helper()
	if len(receiver) > 0 {
		endpointWithReceiver(t, router, http.MethodGet, path, nil, http.StatusOK, nil, receiver[0])
	} else {
		endpoint(t, router, http.MethodGet, path, nil, http.StatusOK, nil)
	}
}

func PutOK(t *testing.T, router http.Handler, path string, body any, receiver ...any) {
	t.Helper()

	if len(receiver) > 0 {
		endpointWithReceiver(t, router, http.MethodPut, path, body, http.StatusOK, nil, receiver[0])
	} else {
		endpoint(t, router, http.MethodPut, path, body, http.StatusOK, nil)
	}
}

func PostAccepted(t *testing.T, router http.Handler, path string, body any, receiver ...any) {
	t.Helper()

	if len(receiver) > 0 {
		endpointWithReceiver(t, router, http.MethodPost, path, body, http.StatusAccepted, nil, receiver[0])
	} else {
		endpoint(t, router, http.MethodPost, path, body, http.StatusAccepted, nil)
	}
}

func PostConflict(t *testing.T, router http.Handler, path string, body any, receiver ...any) {
	t.Helper()

	if len(receiver) > 0 {
		endpointWithReceiver(t, router, http.MethodPost, path, body, http.StatusConflict, nil, receiver[0])
	} else {
		endpoint(t, router, http.MethodPost, path, body, http.StatusConflict, nil)
	}
}

func PostCreatedForm(t *testing.T, router http.Handler, path string, body any, headers map[string]string, receiver ...any) {
	t.Helper()

	if len(receiver) > 0 {
		endpointWithReceiver(t, router, http.MethodPost, path, body, http.StatusCreated, headers, receiver[0])
	} else {
		endpoint(t, router, http.MethodPost, path, body, http.StatusCreated, headers)
	}
}

func PostCreated(t *testing.T, router http.Handler, path string, body any, receiver ...any) {
	t.Helper()

	if len(receiver) > 0 {
		endpointWithReceiver(t, router, http.MethodPost, path, body, http.StatusCreated, nil, receiver[0])
	} else {
		endpoint(t, router, http.MethodPost, path, body, http.StatusCreated, nil)
	}
}

func PostOK(t *testing.T, router http.Handler, path string, body any, receiver ...any) {
	t.Helper()

	if len(receiver) > 0 {
		endpointWithReceiver(t, router, http.MethodPost, path, body, http.StatusOK, nil, receiver[0])
	} else {
		endpoint(t, router, http.MethodPost, path, body, http.StatusOK, nil)
	}
}

func DeleteOK(t *testing.T, router http.Handler, path string, body any, receiver ...any) {
	t.Helper()

	if len(receiver) > 0 {
		endpointWithReceiver(t, router, http.MethodDelete, path, body, http.StatusOK, nil, receiver[0])
	} else {
		endpoint(t, router, http.MethodDelete, path, body, http.StatusOK, nil)
	}
}
func endpointWithReceiver(t *testing.T, router http.Handler, method string,
	path string, body any, expectedStatus int, headers map[string]string, receiver any,
) {
	t.Helper()
	resp := endpoint(t, router, method, path, body, expectedStatus, nil)
	if receiver != nil {
		if err := json.NewDecoder(resp.Body).Decode(&receiver); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
	}
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
