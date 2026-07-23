package httphelper

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

var (
	ErrBadRequest         = errors.New("bad request")
	ErrInternal           = errors.New("internal error")
	ErrNotFound           = errors.New("entity not found")
	ErrRequestPerform     = errors.New("could not perform http request")
	ErrRequestInvalidCode = errors.New("invalid response code returned from request")
	ErrRequestDecode      = errors.New("failed to decode http response")
	ErrRequestCreate      = errors.New("failed to create new request")
	ErrParamKeyMissing    = errors.New("param key not found")
	ErrParamParse         = errors.New("failed to parse param value")
	ErrParamInvalid       = errors.New("param value invalid")
	ErrInvalidParameter   = errors.New("invalid parameter format")
	ErrTooShort           = errors.New("value too short")
	ErrResponseBody       = errors.New("failed to read response body")
)

func NewAPIErrorf(code int, err error, message string, args ...any) APIError {
	apiErr := NewAPIError(code, err)
	apiErr.Detail = fmt.Sprintf(message, args...)

	return apiErr
}

func NewAPIError(code int, err error) APIError {
	apiErr := APIError{
		err:       err,
		Status:    code,
		Type:      "about:blank",
		Timestamp: time.Now(),
	}

	e, ok := err.(interface{ Unwrap() []error })
	if ok {
		wrappedErrs := e.Unwrap()
		if len(wrappedErrs) > 0 {
			apiErr.Title = wrappedErrs[len(wrappedErrs)-1].Error()
		}

		return apiErr
	}

	apiErr.Title = ErrInternal.Error()

	return apiErr
}

type APIError struct {
	err       error
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Status    int       `json:"status"`
	Detail    string    `json:"detail"`
	Instance  string    `json:"instance"`
	Timestamp time.Time `json:"timestamp"`
}

func (e APIError) Error() string {
	if e.err == nil {
		return e.Title
	}

	return e.err.Error()
}

func SetError(w http.ResponseWriter, r *http.Request, err APIError) {
	err.Instance = r.URL.Path
	RespondProblemJSON(w, err.Status, err)
}
