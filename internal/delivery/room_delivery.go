package delivery

import (
	"net/http"

	"github.com/didik-prabowo/sbrm/internal/delivery/middleware"
	ierr "github.com/didik-prabowo/sbrm/internal/domain/error"
	"github.com/didik-prabowo/sbrm/internal/domain/response"
	room "github.com/didik-prabowo/sbrm/internal/domain/room"
	"github.com/didik-prabowo/sbrm/internal/domain/user"
	"github.com/didik-prabowo/sbrm/internal/service"
	"github.com/didik-prabowo/sbrm/pkg/jwt"
	"github.com/labstack/echo/v4"
)

type RoomDelivery struct {
	jwtManager  *jwt.Manager
	RoomService service.RoomService
}

type roomResponse struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
	IsActive *bool  `json:"is_active,omitempty"`
}

func NewRoomDelivery(roomService service.RoomService, jwtManager *jwt.Manager) *RoomDelivery {
	return &RoomDelivery{
		RoomService: roomService,
		jwtManager:  jwtManager,
	}
}

func (dr *RoomDelivery) RegisterRoute(route *echo.Group) {
	auth := middleware.Auth(dr.jwtManager)

	route.POST("/rooms", dr.Create, auth, middleware.RequireRole(user.RoleAdmin))
	route.GET("/rooms/available", dr.Available, auth)
}

func (dr *RoomDelivery) Create(c echo.Context) error {
	ctx := c.Request().Context()

	var req room.CreateRoomInput
	if err := c.Bind(&req); err != nil {
		return ierr.ValidationError("request body tidak valid")
	}

	r, err := dr.RoomService.Create(ctx, middleware.PrincipalFrom(ctx), req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, roomResponse{
		ID:       r.ID,
		Name:     r.Name,
		Capacity: r.Capacity,
		IsActive: &r.IsActive,
	})
}

func (dr *RoomDelivery) Available(c echo.Context) error {
	startStr, endStr := c.QueryParam("start"), c.QueryParam("end")
	if startStr == "" || endStr == "" {
		return ierr.ValidationError("query start dan end wajib diisi")
	}

	start, ok1 := response.ParseTime(startStr)
	end, ok2 := response.ParseTime(endStr)
	if !ok1 || !ok2 {
		return ierr.ErrInvalidTime
	}

	rooms, err := dr.RoomService.FindAvailable(c.Request().Context(), start, end)
	if err != nil {
		return err
	}

	data := make([]roomResponse, 0, len(rooms))
	for _, r := range rooms {
		data = append(data, roomResponse{ID: r.ID, Name: r.Name, Capacity: r.Capacity})
	}

	return c.JSON(http.StatusOK, map[string]any{"data": data})
}
