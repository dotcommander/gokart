package gokart

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ValidatorConfig configures validation behavior.
type ValidatorConfig struct {
	// UseJSONNames uses json tag names in error messages instead of struct field names.
	// Default: true (more useful for API error responses)
	//
	// Note: When true, validation errors report the json tag name (e.g., "email")
	// rather than the struct field name (e.g., "Email"). This is ideal for JSON APIs
	// but may produce confusing messages for CLI applications that use mapstructure
	// or flag tags. Set to false explicitly if you need struct field names.
	UseJSONNames bool
}

// NewValidator creates a configured validator instance.
//
// Default configuration:
//   - Uses JSON tag names for field identification
//   - Registers common custom validators (notblank)
//
// Example:
//
//	v := gokart.NewValidator(gokart.ValidatorConfig{})
//
//	type User struct {
//	    Email string `json:"email" validate:"required,email"`
//	    Age   int    `json:"age" validate:"gte=0,lte=130"`
//	}
//
//	if err := v.Struct(user); err != nil {
//	    // handle validation errors
//	}
func NewValidator(cfg ValidatorConfig) *validator.Validate {
	v := validator.New()

	// Use JSON tag names by default (better for API responses)
	if cfg.UseJSONNames || cfg == (ValidatorConfig{}) {
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			if name == "" {
				return fld.Name
			}
			return name
		})
	}

	// Register notblank: like required but also rejects whitespace-only strings
	_ = v.RegisterValidation("notblank", func(fl validator.FieldLevel) bool {
		field := fl.Field()
		if field.Kind() == reflect.String {
			return strings.TrimSpace(field.String()) != ""
		}
		return field.IsValid() && !field.IsZero()
	})

	return v
}

// NewStandardValidator creates a validator with default settings.
//
// Convenience wrapper around NewValidator with zero configuration.
//
// Example:
//
//	v := gokart.NewStandardValidator()
//	err := v.Struct(myStruct)
func NewStandardValidator() *validator.Validate {
	return NewValidator(ValidatorConfig{})
}

// ValidationErrors extracts field-level errors from a validation error.
// Returns nil if err is not a validator.ValidationErrors.
//
// Example:
//
//	if err := v.Struct(user); err != nil {
//	    for field, msg := range gokart.ValidationErrors(err) {
//	        fmt.Printf("%s: %s\n", field, msg)
//	    }
//	}
func ValidationErrors(err error) map[string]string {
	ve, ok := err.(validator.ValidationErrors)
	if !ok {
		return nil
	}

	errors := make(map[string]string, len(ve))
	for _, fe := range ve {
		errors[fe.Field()] = validationMessage(fe)
	}
	return errors
}

// validationMessage returns a human-readable message for a field error.
func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "notblank":
		return "cannot be blank"
	case "email":
		return "must be a valid email"
	case "url":
		return "must be a valid URL"
	case "uuid":
		return "must be a valid UUID"
	case "min":
		return "is too short"
	case "max":
		return "is too long"
	case "gte":
		return "must be greater than or equal to " + fe.Param()
	case "lte":
		return "must be less than or equal to " + fe.Param()
	case "oneof":
		return "must be one of: " + fe.Param()
	default:
		return "failed " + fe.Tag() + " validation"
	}
}
