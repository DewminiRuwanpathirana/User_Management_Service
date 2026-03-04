package validation

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

var phoneRegex = regexp.MustCompile(`^\+?[0-9][0-9\s\-()]{6,19}$`)

func RegisterPhone(v *validator.Validate) error {
	return v.RegisterValidation("phone", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		if value == "" {
			return true
		}
		return phoneRegex.MatchString(value)
	})
}
