package service

import (
	"context"
	"strings"
	"time"

	ierr "github.com/didik-prabowo/sbrm/internal/domain/error"
	"github.com/didik-prabowo/sbrm/internal/domain/room"
	"github.com/didik-prabowo/sbrm/internal/domain/user"
	"github.com/didik-prabowo/sbrm/internal/repository/postgres"
)

type RoomService interface {
	Create(ctx context.Context, actor user.Principal, in room.CreateRoomInput) (room.Room, error)
	FindAvailable(ctx context.Context, start, end time.Time) ([]room.Room, error)
}

type roomServiceImpl struct {
	roomRepo postgres.RoomRepository
}

func NewRoomService(roomRepo postgres.RoomRepository) RoomService {
	return &roomServiceImpl{
		roomRepo: roomRepo,
	}
}

func (s *roomServiceImpl) Create(ctx context.Context, actor user.Principal, in room.CreateRoomInput) (room.Room, error) {
	if !actor.IsAdmin() {
		return room.Room{}, ierr.ErrForbidden
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len(in.Name) > 100 {
		return room.Room{}, ierr.ValidationError("name wajib diisi, maksimal 100 karakter")
	}
	if in.Capacity <= 0 {
		return room.Room{}, ierr.ValidationError("capacity minimal 1")
	}

	return s.roomRepo.Create(ctx, in.Name, in.Capacity)
}

func (s *roomServiceImpl) FindAvailable(ctx context.Context, start, end time.Time) ([]room.Room, error) {
	if !end.After(start) {
		return nil, ierr.ErrInvalidTime
	}
	return s.roomRepo.FindAvailable(ctx, start, end)
}
