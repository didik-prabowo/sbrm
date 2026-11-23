package middleware

import (
	"context"
	"strings"

	ierr "github.com/didik-prabowo/sbrm/internal/domain/error"
	"github.com/didik-prabowo/sbrm/internal/domain/user"
	"github.com/didik-prabowo/sbrm/pkg/jwt"
	"github.com/labstack/echo/v4"
)

type ctxKey struct{}

type TokenParser interface {
	Parse(token string) (jwt.Claims, error)
}

func Auth(tokens TokenParser) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token, ok := bearerToken(c.Request().Header.Get(echo.HeaderAuthorization))
			if !ok {
				return ierr.ErrUnauthorized
			}

			claims, err := tokens.Parse(token)
			if err != nil {
				return ierr.ErrUnauthorized
			}

			role := user.Role(claims.Role)
			if role != user.RoleAdmin && role != user.RoleUser {
				return ierr.ErrUnauthorized
			}

			req := c.Request()
			ctx := context.WithValue(req.Context(), ctxKey{}, user.Principal{UserID: claims.UserID, Role: role})
			c.SetRequest(req.WithContext(ctx))

			return next(c)
		}
	}
}

func RequireRole(roles ...user.Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			p := PrincipalFrom(c.Request().Context())
			for _, r := range roles {
				if p.Role == r {
					return next(c)
				}
			}
			return ierr.ErrForbidden
		}
	}
}

func PrincipalFrom(ctx context.Context) user.Principal {
	p, _ := ctx.Value(ctxKey{}).(user.Principal)
	return p
}

func bearerToken(header string) (string, bool) {
	scheme, token, ok := strings.Cut(header, " ")
	token = strings.TrimSpace(token)
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", false
	}
	return token, true
}
