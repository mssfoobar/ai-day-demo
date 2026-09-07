package aohhttp

import (
	"errors"
	"net/http"
	"reflect"
	"sync"

	"github.com/go-chi/chi/v5"
	v "github.com/go-playground/validator/v10"
)

const jsonTag string = "json"

var ErrInvalidObj error = errors.New("obj must be a pointer")
var ErrInvalidField error = errors.New("struct field type must be string")

var (
	validatorInit sync.Once
	validator     *v.Validate
)

func newValidator() {
	validator = v.New()
}

// ValidateUriParams validates and stores http request uri parameter in the value pointed to by obj.
// If obj is nil or not a pointer, ValidateUriParmas return [ErrInvalidObj].
func ValidateUriParams(r *http.Request, obj any) error {
	validatorInit.Do(newValidator)

	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr && val.Elem().Kind() == reflect.Struct {
		val = val.Elem()
	} else {
		return ErrInvalidObj
	}

	for i := 0; i < val.NumField(); i++ {
		value := val.Field(i)
		switch value.Kind() {
		case reflect.String:
			typeField := val.Type().Field(i)
			tag := typeField.Tag.Get(jsonTag)
			u := chi.URLParam(r, tag)
			value.Set(reflect.ValueOf(u))
		default:
			return ErrInvalidField
		}
	}

	if err := validator.Struct(obj); err != nil {
		return err
	}
	return nil
}
