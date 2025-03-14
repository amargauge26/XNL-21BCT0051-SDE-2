package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type JWTService struct {
	secretKey []byte
	issuer    string
}

func NewJWTService(secretKey string, issuer string) *JWTService {
	return &JWTService{
		secretKey: []byte(secretKey),
		issuer:    issuer,
	}
}

func (s *JWTService) GenerateToken(userID, role string, duration time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    s.issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// Role-based access control (RBAC)
const (
	RoleAdmin   = "admin"
	RoleTrader  = "trader"
	RoleAnalyst = "analyst"
)

// HasRole checks if the user has the required role
func (c *Claims) HasRole(role string) bool {
	return c.Role == role
}

// HasAnyRole checks if the user has any of the required roles
func (c *Claims) HasAnyRole(roles ...string) bool {
	for _, role := range roles {
		if c.Role == role {
			return true
		}
	}
	return false
}

// IsAdmin checks if the user is an admin
func (c *Claims) IsAdmin() bool {
	return c.Role == RoleAdmin
}

// IsTrader checks if the user is a trader
func (c *Claims) IsTrader() bool {
	return c.Role == RoleTrader
}

// IsAnalyst checks if the user is an analyst
func (c *Claims) IsAnalyst() bool {
	return c.Role == RoleAnalyst
} 