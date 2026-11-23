package jwt

import (
	"errors"
	"strconv"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
)

type JWTConfig struct {
	JWTSecret string        `envconfig:"SECRET" required:"true"`
	JWTTTL    time.Duration `envconfig:"TTL" default:"1h"`
}

type Claims struct {
	UserID int64  `json:"uid"`
	Role   string `json:"role"`
	gojwt.RegisteredClaims
}

type Manager struct {
	secret []byte
	ttl    time.Duration
}

func NewManager(cfg *JWTConfig) *Manager {
	return &Manager{secret: []byte(cfg.JWTSecret), ttl: cfg.JWTTTL}
}

func (m *Manager) Sign(userID int64, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: gojwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			IssuedAt:  gojwt.NewNumericDate(now),
			ExpiresAt: gojwt.NewNumericDate(now.Add(m.ttl)),
		},
	}
	return gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims).SignedString(m.secret)
}

func (m *Manager) Parse(token string) (Claims, error) {
	var claims Claims
	_, err := gojwt.ParseWithClaims(token, &claims, func(*gojwt.Token) (any, error) {
		return m.secret, nil
	}, gojwt.WithValidMethods([]string{gojwt.SigningMethodHS256.Alg()}), gojwt.WithExpirationRequired())
	if err != nil {
		return Claims{}, err
	}
	if claims.UserID <= 0 {
		return Claims{}, errors.New("jwt: missing uid")
	}
	return claims, nil
}
