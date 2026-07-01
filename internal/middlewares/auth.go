package middlewares

import (
	"job-agent-api/internal/services"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

func JWTMiddleware(next echo.HandlerFunc) echo.HandlerFunc {

	return func(c *echo.Context) error {

		authHeader := c.Request().Header.Get("Authorization")

		if authHeader == "" {
			return echo.NewHTTPError(
				http.StatusUnauthorized,
				"Token ausente",
			)
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.ParseWithClaims(
			tokenString,
			&services.Claims{},
			func(token *jwt.Token) (interface{}, error) {

				return []byte(os.Getenv("JWT_SECRET")), nil
			},
		)

		if err != nil {
			return echo.NewHTTPError(
				http.StatusUnauthorized,
				"Token inválido",
			)
		}

		claims, ok := token.Claims.(*services.Claims)

		if time.Now().Compare(claims.ExpiresAt.Time) == 1  {
			return echo.NewHTTPError(
				http.StatusUnauthorized,
				"Token expirado.",
			)
		}

		if !ok || !token.Valid {
			return echo.NewHTTPError(
				http.StatusUnauthorized,
				"Token inválido",
			)
		}

		c.Set("user", claims)

		return next(c)
	}

}