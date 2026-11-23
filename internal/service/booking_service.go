package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/didik-prabowo/sbrm/internal/domain/booking"
	ierr "github.com/didik-prabowo/sbrm/internal/domain/error"
	"github.com/didik-prabowo/sbrm/internal/domain/user"
	"github.com/didik-prabowo/sbrm/internal/repository/postgres"
)

type Transactor interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type BookingService interface {
	Create(ctx context.Context, actor user.Principal, in booking.CreateBookingInput) (booking.Booking, error)
	Cancel(ctx context.Context, actor user.Principal, bookingID int64) (booking.Booking, error)
}

type bookingServiceImpl struct {
	tx          Transactor
	roomRepo    postgres.RoomRepository
	bookingRepo postgres.BookingRepository
	now         func() time.Time
}

func NewBookingService(tx Transactor, roomRepo postgres.RoomRepository, bookingRepo postgres.BookingRepository) BookingService {
	return &bookingServiceImpl{
		tx:          tx,
		roomRepo:    roomRepo,
		bookingRepo: bookingRepo,
		now:         time.Now,
	}
}

func (s *bookingServiceImpl) Create(ctx context.Context, actor user.Principal, in booking.CreateBookingInput) (booking.Booking, error) {
	in.Title = strings.TrimSpace(in.Title)
	if in.RoomID <= 0 {
		return booking.Booking{}, ierr.ValidationError("room_id wajib diisi")
	}
	if in.Title == "" || len(in.Title) > 150 {
		return booking.Booking{}, ierr.ValidationError("title wajib diisi, maksimal 150 karakter")
	}
	if !in.EndTime.After(in.StartTime) || in.StartTime.Before(s.now()) {
		return booking.Booking{}, ierr.ErrInvalidTime
	}

	var created booking.Booking
	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		r, err := s.roomRepo.LockByID(ctx, in.RoomID)
		if errors.Is(err, ierr.ErrRecordNotFound) {
			return ierr.ErrRoomNotFound
		}
		if err != nil {
			return err
		}
		if !r.IsActive {
			return ierr.ErrRoomInactive
		}

		overlap, err := s.bookingRepo.HasOverlap(ctx, r.ID, in.StartTime, in.EndTime)
		if err != nil {
			return err
		}
		if overlap {
			return ierr.ErrRoomConflict
		}

		created, err = s.bookingRepo.Create(ctx, booking.Booking{
			RoomID:    r.ID,
			UserID:    actor.UserID,
			Title:     in.Title,
			StartTime: in.StartTime,
			EndTime:   in.EndTime,
			Status:    booking.StatusConfirmed,
		})
		return err
	})

	return created, err
}

func (s *bookingServiceImpl) Cancel(ctx context.Context, actor user.Principal, bookingID int64) (booking.Booking, error) {
	var cancelled booking.Booking
	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		b, err := s.bookingRepo.LockByID(ctx, bookingID)
		if errors.Is(err, ierr.ErrRecordNotFound) {
			return ierr.ErrBookingNotFound
		}
		if err != nil {
			return err
		}
		if b.UserID != actor.UserID && !actor.IsAdmin() {
			return ierr.ErrForbidden
		}
		if b.Status == booking.StatusCancelled {
			return ierr.ErrAlreadyCancelled
		}

		cancelled, err = s.bookingRepo.UpdateStatus(ctx, b.ID, booking.StatusCancelled)
		return err
	})

	return cancelled, err
}
