package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-playground/validator/v10"
)

func respondInvalidRequest(w http.ResponseWriter, err error) {
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		writeValidationError(w, "validation failed", validationFieldMap(err))
		return
	}
	var se *json.SyntaxError
	var te *json.UnmarshalTypeError
	switch {
	case errors.Is(err, io.EOF):
		writeError(w, http.StatusBadRequest, "empty request body")
	case errors.As(err, &se), errors.As(err, &te):
		writeError(w, http.StatusBadRequest, "invalid json")
	default:
		writeError(w, http.StatusBadRequest, "invalid request body")
	}
}
