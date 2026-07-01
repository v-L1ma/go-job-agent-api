package services

import (
	"job-agent-api/internal/database/sqlc"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		12,
	)

	return string(bytes), err
}

func CheckPassword(password string, hash string) error {
    return bcrypt.CompareHashAndPassword(
        []byte(hash),
        []byte(password),
    )
}

type Claims struct {
    UserID string `json:"user_id"`
    Email  string `json:"email"`

    jwt.RegisteredClaims
}

func GenerateToken(user sqlc.AspNetUser) (string, error) {

    claims := Claims{
        UserID: user.Id.String(),
        Email: user.Email.String,

        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(
                time.Now().Add(24 * time.Hour),
            ),
            IssuedAt: jwt.NewNumericDate(time.Now()),
        },
    }

    token := jwt.NewWithClaims(
        jwt.SigningMethodHS256,
        claims,
    )

    return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}