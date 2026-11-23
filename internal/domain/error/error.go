package error

import (
	"errors"
	"net/http"
)

type AppError struct {
	HTTPStatus int
	Code       string
	Message    string
}

func (e *AppError) Error() string { return e.Code + ": " + e.Message }

func NewError(status int, code, msg string) *AppError {
	return &AppError{HTTPStatus: status, Code: code, Message: msg}
}

func AsAppError(err error) (*AppError, bool) {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}

var (
	ErrInvalidCredentials = NewError(http.StatusUnauthorized, "INVALID_CREDENTIALS", "kombinasi email dan password tidak cocok")
	ErrUnauthorized       = NewError(http.StatusUnauthorized, "UNAUTHORIZED", "silakan login terlebih dahulu")
	ErrForbidden          = NewError(http.StatusForbidden, "FORBIDDEN", "akses ditolak")
	ErrInvalidTime        = NewError(http.StatusBadRequest, "INVALID_TIME", "waktu tidak valid, pastikan format RFC3339, end_time > start_time, dan start_time belum lewat")
	ErrRoomNotFound       = NewError(http.StatusNotFound, "ROOM_NOT_FOUND", "room tidak ada")
	ErrRoomInactive       = NewError(http.StatusBadRequest, "ROOM_INACTIVE", "room sedang nonaktif")
	ErrRoomConflict       = NewError(http.StatusConflict, "ROOM_CONFLICT", "jadwal bentrok dengan booking lain di room ini")
	ErrBookingNotFound    = NewError(http.StatusNotFound, "BOOKING_NOT_FOUND", "booking tidak ada")
	ErrAlreadyCancelled   = NewError(http.StatusConflict, "BOOKING_ALREADY_CANCELLED", "booking ini sudah dibatalkan")
	ErrNotFound           = NewError(http.StatusNotFound, "NOT_FOUND", "route tidak ditemukan")
	ErrMethodNotAllowed   = NewError(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method tidak didukung")
	ErrInternal           = NewError(http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
)

func ValidationError(msg string) *AppError {
	return NewError(http.StatusBadRequest, "VALIDATION_ERROR", msg)
}

var ErrRecordNotFound = errors.New("record not found")
