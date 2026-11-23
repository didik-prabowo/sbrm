package booking

import "time"

type Status string

const (
	StatusConfirmed Status = "CONFIRMED"
	StatusCancelled Status = "CANCELLED"
)

type Booking struct {
	ID        int64
	RoomID    int64
	UserID    int64
	Title     string
	StartTime time.Time
	EndTime   time.Time
	Status    Status
}

type CreateBookingInput struct {
	RoomID    int64
	Title     string
	StartTime time.Time
	EndTime   time.Time
}
