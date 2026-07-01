package helpers

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

func ValidationErrors(err error) map[string]string {
	errors := make(map[string]string)

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		errors["error"] = "Erro de validação"
		return errors
	}

	for _, e := range validationErrors {
		field := strings.ToLower(e.Field())

		switch e.Tag() {
		case "required":
			errors[field] = "Este campo é obrigatório"

		case "email":
			errors[field] = "E-mail inválido"

		case "min":
			errors[field] = fmt.Sprintf("Deve possuir no mínimo %s caracteres", e.Param())

		case "max":
			errors[field] = fmt.Sprintf("Deve possuir no máximo %s caracteres", e.Param())

		case "oneof":
			errors[field] = fmt.Sprintf("Valor deve ser um dos seguintes: %s", e.Param())

		case "uuid":
			errors[field] = "UUID inválido"

		default:
			errors[field] = "Valor inválido"
		}
	}

	return errors
}