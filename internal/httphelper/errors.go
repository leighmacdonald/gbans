package httphelper

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
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
	} else {
		apiErr.Title = err.Error()
	}

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

func SetError(ctx *gin.Context, error APIError) {
	error.Instance = ctx.Request.URL.Path

	_ = ctx.Error(error)
}
