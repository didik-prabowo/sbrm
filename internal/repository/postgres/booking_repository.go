package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/didik-prabowo/sbrm/internal/domain/booking"
	ierr "github.com/didik-prabowo/sbrm/internal/domain/error"
	"github.com/jackc/pgconn"
)

const pgExclusionViolation = "23P01"

type BookingRepository interface {
	HasOverlap(ctx context.Context, roomID int64, start, end time.Time) (bool, error)
	Create(ctx context.Context, b booking.Booking) (booking.Booking, error)
	LockByID(ctx context.Context, id int64) (booking.Booking, error)
	UpdateStatus(ctx context.Context, id int64, status booking.Status) (booking.Booking, error)
}

type bookingRepositoryImpl struct {
	sqlRepo *SQLRepository
}

func NewBookingRepository(sqlRepo *SQLRepository) BookingRepository {
	return &bookingRepositoryImpl{
		sqlRepo: sqlRepo,
	}
}

const bookingColumns = `id, room_id, user_id, title, start_time, end_time, status`

func scanBooking(row *sql.Row) (booking.Booking, error) {
	var (
		b      booking.Booking
		status string
	)
	if err := row.Scan(&b.ID, &b.RoomID, &b.UserID, &b.Title, &b.StartTime, &b.EndTime, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return b, ierr.ErrRecordNotFound
		}
		return b, err
	}
	b.Status = booking.Status(status)
	b.StartTime = b.StartTime.UTC()
	b.EndTime = b.EndTime.UTC()
	return b, nil
}

func (repo *bookingRepositoryImpl) HasOverlap(ctx context.Context, roomID int64, start, end time.Time) (bool, error) {
	var exists bool
	err := repo.sqlRepo.conn(ctx).QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM bookings
			WHERE room_id = $1
			  AND status = 'CONFIRMED'
			  AND start_time < $3
			  AND end_time > $2)`, roomID, start.UTC(), end.UTC(),
	).Scan(&exists)
	return exists, err
}

func (repo *bookingRepositoryImpl) Create(ctx context.Context, b booking.Booking) (booking.Booking, error) {
	row := repo.sqlRepo.conn(ctx).QueryRowContext(ctx,
		`INSERT INTO bookings (room_id, user_id, title, start_time, end_time, status)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING `+bookingColumns,
		b.RoomID, b.UserID, b.Title, b.StartTime.UTC(), b.EndTime.UTC(), string(b.Status),
	)

	created, err := scanBooking(row)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgExclusionViolation {
		return booking.Booking{}, ierr.ErrRoomConflict
	}
	return created, err
}

func (repo *bookingRepositoryImpl) LockByID(ctx context.Context, id int64) (booking.Booking, error) {
	return scanBooking(repo.sqlRepo.conn(ctx).QueryRowContext(ctx,
		`SELECT `+bookingColumns+` FROM bookings WHERE id = $1 FOR UPDATE`, id))
}

func (repo *bookingRepositoryImpl) UpdateStatus(ctx context.Context, id int64, status booking.Status) (booking.Booking, error) {
	return scanBooking(repo.sqlRepo.conn(ctx).QueryRowContext(ctx,
		`UPDATE bookings SET status = $2, updated_at = NOW() AT TIME ZONE 'UTC'
		 WHERE id = $1
		 RETURNING `+bookingColumns, id, string(status)))
}
