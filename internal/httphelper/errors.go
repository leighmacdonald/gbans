package httphelper

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	ErrBadRequest         = errors.New("invalid request")
	ErrInternal           = errors.New("internal server error")
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
		// Error was wrapped with errors.Join(), so we want to only show the very last error, which should be one of our
		// common sentinel errors that is safe for showing and wont expose any internal details.
		wrappedErrs := e.Unwrap()
		if len(wrappedErrs) > 0 {
			apiErr.Title = wrappedErrs[len(wrappedErrs)-1].Error()
		}

		return apiErr
	}

	apiErr.Title = err.Error()

	return apiErr
}

// APIError implements https://www.rfc-editor.org/rfc/rfc9457.html
// application/problem+json.
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
		// Its just a simple validation error, which does not have any wrapped errors.
		return e.Title
	}

	return e.err.Error()
}

// SetError handles sending the error to the error handler middleware. You should return
// from the handler after calling this.
func SetError(ctx *gin.Context, err APIError) {
	err.Instance = ctx.Request.URL.Path

	_ = ctx.Error(err)
}
