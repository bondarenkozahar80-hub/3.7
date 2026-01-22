package auth

import (
	"errors"
	"time"
	"3.7/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte("your-secret-key-change-in-production")

type Claims struct {
	Username string        `json:"username"`
	Role     models.Role   `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(username string, role models.Role) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func HasPermission(role models.Role, action string) bool {
	permissions := map[models.Role][]string{
		models.RoleAdmin:   {"create", "read", "update", "delete", "history"},
		models.RoleManager: {"create", "read", "update", "history"},
		models.RoleViewer:  {"read"},
		models.RoleAuditor: {"read", "history"},
	}
	for _, perm := range permissions[role] {
		if perm == action {
			return true
		}
	}
	return false
}
