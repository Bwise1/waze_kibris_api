package util

import "github.com/go-playground/validator/v10"

var validate *validator.Validate

func init() {
	validate = validator.New()
	validate.RegisterValidation("latitude", validateLatitude)
	validate.RegisterValidation("longitude", validateLongitude)
}

func validateLatitude(fl validator.FieldLevel) bool {
	lat := fl.Field().Float()
	return lat >= -90 && lat <= 90
}

func validateLongitude(fl validator.FieldLevel) bool {
	lon := fl.Field().Float()
	return lon >= -180 && lon <= 180
}

func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}
