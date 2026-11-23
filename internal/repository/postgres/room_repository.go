package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	ierr "github.com/didik-prabowo/sbrm/internal/domain/error"
	room "github.com/didik-prabowo/sbrm/internal/domain/room"
)

type RoomRepository interface {
	Create(ctx context.Context, name string, capacity int) (room.Room, error)
	LockByID(ctx context.Context, id int64) (room.Room, error)
	FindAvailable(ctx context.Context, start, end time.Time) ([]room.Room, error)
}

type roomRepositoryImpl struct {
	sqlRepo *SQLRepository
}

func NewRoomRepository(sqlRepo *SQLRepository) RoomRepository {
	return &roomRepositoryImpl{
		sqlRepo: sqlRepo,
	}
}

func (repo *roomRepositoryImpl) Create(ctx context.Context, name string, capacity int) (room.Room, error) {
	var r room.Room
	err := repo.sqlRepo.conn(ctx).QueryRowContext(ctx,
		`INSERT INTO rooms (name, capacity) VALUES ($1, $2)
		 RETURNING id, name, capacity, is_active`, name, capacity,
	).Scan(&r.ID, &r.Name, &r.Capacity, &r.IsActive)
	return r, err
}

func (repo *roomRepositoryImpl) LockByID(ctx context.Context, id int64) (room.Room, error) {
	var r room.Room
	err := repo.sqlRepo.conn(ctx).QueryRowContext(ctx,
		`SELECT id, name, capacity, is_active FROM rooms WHERE id = $1 FOR UPDATE`, id,
	).Scan(&r.ID, &r.Name, &r.Capacity, &r.IsActive)
	if errors.Is(err, sql.ErrNoRows) {
		return r, ierr.ErrRecordNotFound
	}
	return r, err
}

func (repo *roomRepositoryImpl) FindAvailable(ctx context.Context, start, end time.Time) ([]room.Room, error) {
	rows, err := repo.sqlRepo.conn(ctx).QueryContext(ctx, `
		SELECT r.id, r.name, r.capacity, r.is_active
		FROM rooms r
		WHERE r.is_active
		  AND NOT EXISTS (
		        SELECT 1 FROM bookings b
		        WHERE b.room_id = r.id
		          AND b.status = 'CONFIRMED'
		          AND b.start_time < $2
		          AND b.end_time > $1)
		ORDER BY r.id`, start.UTC(), end.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rooms := make([]room.Room, 0)
	for rows.Next() {
		var r room.Room
		if err := rows.Scan(&r.ID, &r.Name, &r.Capacity, &r.IsActive); err != nil {
			return nil, err
		}
		rooms = append(rooms, r)
	}
	return rooms, rows.Err()
}
