package api

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/asaskevich/govalidator"
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

func IsValidateError(err error) bool {
	var e *ValidateError
	return errors.As(err, &e)
}

// helper

var reEnvName = regexp.MustCompile(`^[-._a-zA-Z][-._a-zA-Z0-9]*$`)

func validEnvName(env map[string]string) bool {
	for k := range env {
		if !reEnvName.MatchString(k) {
			return false
		}
	}
	return true
}

func validImage(image string) bool {
	if strings.HasSuffix(image, "@") {
		return false
	}

	return true
}

func validRouteTarget(target string) bool {
	for _, x := range routeTargetPrefix {
		if strings.HasPrefix(target, x) {
			return true
		}
	}
	return false
}

func validURL(url string) bool {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return govalidator.IsURL(url)
	}
	return false
}
