package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

var requestValidator = validator.New()

func decodeAndValidate(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return requestValidator.Struct(dst)
}

func validationFieldMap(err error) map[string]string {
	out := make(map[string]string)
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return out
	}
	for _, fe := range verrs {
		out[strings.ToLower(fe.Field())] = fe.Error()
	}
	return out
}
