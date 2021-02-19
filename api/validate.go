package api

import (
	"encoding/json"

	"github.com/moonrhythm/validator"
)

type ValidateError struct {
	err *validator.Error
}

func (err *ValidateError) Error() string {
	return err.err.Error()
}

func (err *ValidateError) OKError() {}

func (err *ValidateError) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Message string   `json:"message"`
		Items   []string `json:"items"`
	}{"api: validate error", err.err.Strings()})
}

func (err *ValidateError) Items() []error {
	return err.err.Errors()
}

func WrapValidate(v *validator.Validator) error {
	if err := v.Error(); err != nil {
		return &ValidateError{err.(*validator.Error)}
	}
	return nil
}
