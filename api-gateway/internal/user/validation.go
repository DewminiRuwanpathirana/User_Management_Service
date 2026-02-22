package user

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

var phoneRegex = regexp.MustCompile(`^\+?[0-9][0-9\s\-()]{6,19}$`)

func NewValidator() *validator.Validate {
	validate := validator.New()

	if err := validate.RegisterValidation("phone", func(fl validator.FieldLevel) bool {
		phone := fl.Field().String()
		if phone == "" {
			return true
		}

		return phoneRegex.MatchString(phone)
	}); err != nil {
		panic(err)
	}

	return validate
}
