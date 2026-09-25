package httputil

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var (
	validate     *validator.Validate
	validateOnce sync.Once
)

func Validator() *validator.Validate {
	validateOnce.Do(func() {
		validate = validator.New(validator.WithRequiredStructEnabled())
		// Error messages use the json field name.
		validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})
	})
	return validate
}

// Does not validate — prefer DecodeAndValidate.
func DecodeJSON(r *http.Request, v interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("request body is empty")
		}
		return err
	}
	return nil
}

func DecodeAndValidate(r *http.Request, v interface{}) error {
	if err := DecodeJSON(r, v); err != nil {
		return NewAPIError(http.StatusBadRequest, "Invalid request body: "+err.Error(), err)
	}
	if err := Validator().Struct(v); err != nil {
		return NewAPIError(http.StatusBadRequest, formatValidationError(err), err)
	}
	return nil
}

func formatValidationError(err error) string {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return "Invalid request: " + err.Error()
	}
	parts := make([]string, 0, len(ve))
	for _, fe := range ve {
		parts = append(parts, fmt.Sprintf("%s: %s", fe.Field(), tagToMessage(fe)))
	}
	return strings.Join(parts, "; ")
}

func tagToMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email"
	case "uuid", "uuid4":
		return "must be a valid UUID"
	case "min":
		return "must be at least " + fe.Param() + " chars"
	case "max":
		return "must be at most " + fe.Param() + " chars"
	case "oneof":
		return "must be one of: " + fe.Param()
	case "gte":
		return "must be ≥ " + fe.Param()
	case "lte":
		return "must be ≤ " + fe.Param()
	case "hexcolor":
		return "must be a valid hex color"
	default:
		return "is invalid (" + fe.Tag() + ")"
	}
}

// Format check only; sqlc uses string ids.
func GetUUIDParam(r *http.Request, key string) (string, error) {
	paramStr := chi.URLParam(r, key)
	if paramStr == "" {
		return "", errors.New("missing parameter: " + key)
	}
	if _, err := uuid.Parse(paramStr); err != nil {
		return "", errors.New("invalid UUID format for parameter: " + key)
	}
	return paramStr, nil
}
