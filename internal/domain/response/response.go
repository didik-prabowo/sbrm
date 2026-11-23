package response

import (
	"time"

	ierr "github.com/didik-prabowo/sbrm/internal/domain/error"
)

type errorResponse struct {
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
}

func Error(err error) errorResponse {
	if ae, ok := ierr.AsAppError(err); ok {
		return errorResponse{ErrorCode: ae.Code, Message: ae.Message}
	}

	return errorResponse{ErrorCode: ierr.ErrInternal.Code, Message: ierr.ErrInternal.Message}
}

func FormatTime(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func ParseTime(s string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, false
	}
	return t.UTC(), true
}
