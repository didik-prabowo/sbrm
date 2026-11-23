package delivery

import (
	"net/http"

	ierr "github.com/didik-prabowo/sbrm/internal/domain/error"
	"github.com/didik-prabowo/sbrm/internal/service"
	"github.com/labstack/echo/v4"
)

type AuthDelivery struct {
	AuthService service.AuthService
}

func NewAuthDelivery(authService service.AuthService) *AuthDelivery {
	return &AuthDelivery{
		AuthService: authService,
	}
}

func (ad *AuthDelivery) RegisterRoute(route *echo.Group) {
	route.POST("/auth/login", ad.Login)
}

func (ad *AuthDelivery) Login(c echo.Context) error {
	type loginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	form := new(loginRequest)
	if err := c.Bind(form); err != nil {
		return ierr.ValidationError("request body tidak valid")
	}

	token, err := ad.AuthService.Login(c.Request().Context(), form.Email, form.Password)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]string{"token": token})
}
