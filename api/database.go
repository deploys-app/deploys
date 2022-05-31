package api

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/moonrhythm/validator"
)

type Database interface {
	Create(context.Context, *DatabaseCreate) (*Empty, error)
}

type DatabaseCreate struct {
	Project  string
	Location string
	Name     string
}

func (m *DatabaseCreate) Valid() error {
	m.Name = strings.TrimSpace(m.Name)

	v := validator.New()

	v.Must(m.Project != "", "project required")
	v.Must(m.Location != "", "location required")
	v.Must(ReValidName.MatchString(m.Name), "name invalid "+ReValidNameStr)
	{
		cnt := utf8.RuneCountInString(m.Name)
		v.Mustf(cnt >= MinNameLength && cnt <= MaxNameLength, "name must have length between %d-%d characters", MinNameLength, MaxNameLength)
	}

	return WrapValidate(v)
}

type DatabaseList struct {
	Project  string `json:"project" yaml:"project"`
	Location string `json:"location" yaml:"location"`
}

func (m *DatabaseList) Valid() error {
	v := validator.New()

	v.Must(m.Project != "", "project required")

	return WrapValidate(v)
}

type DatabaseListResult struct {
	Items []*DatabaseItem
}

type DatabaseItem struct {
	ID        int64
	ProjectID int64
	Location  string
	Name      string
	Status    Status
	Action    string
	CreatedAt time.Time
	CreatedBy string
	SuccessAt time.Time
}

type DatabaseDelete struct {
	Project string
	Name    string
}

func (m *DatabaseDelete) Valid() error {
	m.Name = strings.TrimSpace(m.Name)

	v := validator.New()

	v.Must(ReValidName.MatchString(m.Name), "name invalid "+ReValidNameStr)
	if cnt := utf8.RuneCountInString(m.Name); cnt > MaxNameLength {
		return fmt.Errorf("name invalid")
	}
	v.Must(m.Project != "", "project required")

	return WrapValidate(v)
}
