package delivery

import (
	"errors"
	"log"
	"net/http"

	ierr "github.com/didik-prabowo/sbrm/internal/domain/error"
	"github.com/didik-prabowo/sbrm/internal/domain/response"
	"github.com/labstack/echo/v4"
)

func HTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	appErr, ok := ierr.AsAppError(err)
	if !ok {
		appErr = fromEchoError(err)
	}
	if appErr.HTTPStatus >= http.StatusInternalServerError {
		log.Printf("ERROR: %s %s: %v", c.Request().Method, c.Request().URL.Path, err)
	}

	if err := c.JSON(appErr.HTTPStatus, response.Error(appErr)); err != nil {
		log.Printf("ERROR: write error response: %v", err)
	}
}

func fromEchoError(err error) *ierr.AppError {
	var he *echo.HTTPError
	if !errors.As(err, &he) {
		return ierr.ErrInternal
	}

	switch he.Code {
	case http.StatusNotFound:
		return ierr.ErrNotFound
	case http.StatusMethodNotAllowed:
		return ierr.ErrMethodNotAllowed
	case http.StatusBadRequest, http.StatusUnsupportedMediaType:
		return ierr.ValidationError("request body tidak valid")
	case http.StatusUnauthorized:
		return ierr.ErrUnauthorized
	case http.StatusForbidden:
		return ierr.ErrForbidden
	}
	return ierr.ErrInternal
}
