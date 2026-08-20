package middlewares

import (
	"fmt"
	"job-agent-api/internal/services"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

func AdminMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
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
				// Garante que o algoritmo é o esperado
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("algoritmo inválido")
				}

				return []byte(os.Getenv("JWT_SECRET")), nil
			},
		)

		if err != nil {
			return echo.NewHTTPError(
				http.StatusUnauthorized,
				"Token inválido",
			)
		}

		if !token.Valid {
			return echo.NewHTTPError(
				http.StatusUnauthorized,
				"Token inválido",
			)
		}

		claims, ok := token.Claims.(*services.Claims)
		if !ok {
			return echo.NewHTTPError(
				http.StatusUnauthorized,
				"Claims inválidas",
			)
		}

		if strings.ToUpper(claims.Role) != "ADMIN" {
			return echo.NewHTTPError(
				http.StatusForbidden,
				"Acesso negado",
			)
		}

		c.Set("user", claims)

		return next(c)
	}
}