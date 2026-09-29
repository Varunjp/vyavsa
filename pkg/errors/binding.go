package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

// ParseBindingError inspects errors originating from JSON unmarshaling and struct validation
// to produce a clean, production-grade *AppError with proper status codes and structured field errors.
func ParseBindingError(err error) *AppError {
	if err == nil {
		return nil
	}

	// 1. Empty body
	if errors.Is(err, io.EOF) {
		return NewBadRequest("request body cannot be empty", err)
	}

	// 2. Malformed JSON syntax
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return NewBadRequest(fmt.Sprintf("malformed JSON at byte offset %d", syntaxErr.Offset), err)
	}

	// 3. Type mismatch during unmarshaling
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		field := typeErr.Field
		if field == "" {
			field = "payload"
		}
		fieldKey := toSnakeCase(field)
		msg := fmt.Sprintf("invalid type for field '%s': expected %s, got %s", fieldKey, typeErr.Type.String(), typeErr.Value)
		return NewValidation("invalid request payload", map[string]string{fieldKey: msg}, err)
	}

	// 4. Structured field-level validation errors (go-playground/validator)
	var valErrors validator.ValidationErrors
	if errors.As(err, &valErrors) {
		fields := make(map[string]string, len(valErrors))
		for _, f := range valErrors {
			fieldName := toSnakeCase(f.Field())
			switch f.Tag() {
			case "required":
				fields[fieldName] = fmt.Sprintf("%s is required", fieldName)
			case "email":
				fields[fieldName] = fmt.Sprintf("%s must be a valid email address", fieldName)
			case "min":
				fields[fieldName] = fmt.Sprintf("%s must be at least %s characters", fieldName, f.Param())
			case "max":
				fields[fieldName] = fmt.Sprintf("%s must be at most %s characters", fieldName, f.Param())
			case "oneof":
				fields[fieldName] = fmt.Sprintf("%s must be one of: %s", fieldName, f.Param())
			case "gt":
				fields[fieldName] = fmt.Sprintf("%s must be greater than %s", fieldName, f.Param())
			case "gte":
				fields[fieldName] = fmt.Sprintf("%s must be greater than or equal to %s", fieldName, f.Param())
			case "lt":
				fields[fieldName] = fmt.Sprintf("%s must be less than %s", fieldName, f.Param())
			case "lte":
				fields[fieldName] = fmt.Sprintf("%s must be less than or equal to %s", fieldName, f.Param())
			case "uuid":
				fields[fieldName] = fmt.Sprintf("%s must be a valid UUID", fieldName)
			default:
				fields[fieldName] = fmt.Sprintf("%s failed validation rule '%s'", fieldName, f.Tag())
			}
		}
		return NewValidation("validation failed", fields, err)
	}

	// 5. Fallback for other binding or unmarshaling errors
	return NewValidation("invalid request payload", map[string]string{
		"error": err.Error(),
	}, err)
}

func toSnakeCase(str string) string {
	if str == "" {
		return ""
	}
	var b strings.Builder
	for i, r := range str {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
