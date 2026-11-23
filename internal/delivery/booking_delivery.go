package delivery

import (
	"net/http"
	"strconv"

	"github.com/didik-prabowo/sbrm/internal/delivery/middleware"
	"github.com/didik-prabowo/sbrm/internal/domain/booking"
	ierr "github.com/didik-prabowo/sbrm/internal/domain/error"
	"github.com/didik-prabowo/sbrm/internal/domain/response"
	"github.com/didik-prabowo/sbrm/internal/service"
	"github.com/didik-prabowo/sbrm/pkg/jwt"
	"github.com/labstack/echo/v4"
)

type BookingDelivery struct {
	jwtManager     *jwt.Manager
	BookingService service.BookingService
}

type bookingResponse struct {
	ID        int64  `json:"id"`
	RoomID    int64  `json:"room_id"`
	UserID    int64  `json:"user_id"`
	Title     string `json:"title"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Status    string `json:"status"`
}

func NewBookingDelivery(bookingService service.BookingService, jwtManager *jwt.Manager) *BookingDelivery {
	return &BookingDelivery{
		BookingService: bookingService,
		jwtManager:     jwtManager,
	}
}

func (bd *BookingDelivery) RegisterRoute(route *echo.Group) {
	auth := middleware.Auth(bd.jwtManager)

	route.POST("/bookings", bd.Create, auth)
	route.PATCH("/bookings/:id/cancel", bd.Cancel, auth)
}

func (bd *BookingDelivery) Create(c echo.Context) error {
	ctx := c.Request().Context()

	var req struct {
		RoomID    int64  `json:"room_id"`
		Title     string `json:"title"`
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
	}
	if err := c.Bind(&req); err != nil {
		return ierr.ValidationError("request body tidak valid")
	}
	if req.RoomID == 0 || req.Title == "" || req.StartTime == "" || req.EndTime == "" {
		return ierr.ValidationError("room_id, title, start_time, dan end_time wajib diisi")
	}

	start, ok1 := response.ParseTime(req.StartTime)
	end, ok2 := response.ParseTime(req.EndTime)
	if !ok1 || !ok2 {
		return ierr.ErrInvalidTime
	}

	b, err := bd.BookingService.Create(ctx, middleware.PrincipalFrom(ctx), booking.CreateBookingInput{
		RoomID:    req.RoomID,
		Title:     req.Title,
		StartTime: start,
		EndTime:   end,
	})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, bookingResponse{
		ID:        b.ID,
		RoomID:    b.RoomID,
		UserID:    b.UserID,
		Title:     b.Title,
		StartTime: response.FormatTime(b.StartTime),
		EndTime:   response.FormatTime(b.EndTime),
		Status:    string(b.Status),
	})
}

func (bd *BookingDelivery) Cancel(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return ierr.ErrBookingNotFound
	}

	b, err := bd.BookingService.Cancel(ctx, middleware.PrincipalFrom(ctx), id)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]any{"id": b.ID, "status": b.Status})
}
