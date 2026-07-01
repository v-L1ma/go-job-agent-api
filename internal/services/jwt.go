package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
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
                time.Now().Add(15 * time.Minute),
            ),
            IssuedAt: jwt.NewNumericDate(time.Now()),
        },
    }

    return signClaims(claims)
}

func GenerateTokenFromClaims(claims *Claims) (string, error) {
    claims.ExpiresAt = jwt.NewNumericDate(
        time.Now().Add(15 * time.Hour),
    )
    claims.IssuedAt = jwt.NewNumericDate(time.Now())

    return signClaims(*claims)
}

func signClaims(claims Claims) (string, error) {
    token := jwt.NewWithClaims(
        jwt.SigningMethodHS256,
        claims,
    )

    return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}

func GenerateRandomToken(length int) (string, error) {
    bytes := make([]byte, length)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return hex.EncodeToString(bytes), nil
}

func HashToken(token string) string {
    hash := sha256.Sum256([]byte(token))
    return hex.EncodeToString(hash[:])
}