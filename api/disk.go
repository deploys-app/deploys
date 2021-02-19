package api

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/moonrhythm/validator"
)

type Disk interface {
	Create(ctx context.Context, m DiskCreate) (*Empty, error)
	List(ctx context.Context, m DiskList) (*DiskListResult, error)
	Update(ctx context.Context, m DiskUpdate) (*Empty, error)
	Delete(ctx context.Context, m DiskDelete) (*Empty, error)
}

type DiskCreate struct {
	Project  string
	Location string
	Name     string
	Size     int64
}

func (m *DiskCreate) Valid() error {
	m.Name = strings.TrimSpace(m.Name)

	v := validator.New()

	v.Must(m.Project != "", "project required")
	v.Must(m.Location != "", "location required")
	v.Must(ReValidName.MatchString(m.Name), "name invalid "+ReValidNameStr)
	{
		cnt := utf8.RuneCountInString(m.Name)
		v.Mustf(cnt >= MinNameLength && cnt <= MaxNameLength, "name must have length between %d-%d characters", MinNameLength, MaxNameLength)
	}
	v.Must(m.Size >= 1, "minimum disk size 1 Gi")
	v.Must(m.Size <= 20, "maximum disk size 20 Gi")

	return WrapValidate(v)
}

type DiskUpdate struct {
	Project string
	Name    string
	Size    int64
}

func (m *DiskUpdate) Valid() error {
	m.Name = strings.TrimSpace(m.Name)

	v := validator.New()

	v.Must(m.Project != "", "project required")
	v.Must(ReValidName.MatchString(m.Name), "name invalid "+ReValidNameStr)
	{
		cnt := utf8.RuneCountInString(m.Name)
		v.Mustf(cnt >= MinNameLength && cnt <= MaxNameLength, "name must have length between %d-%d characters", MinNameLength, MaxNameLength)
	}
	v.Must(m.Size >= 1, "minimum disk size 1 Gi")
	v.Must(m.Size <= 20, "maximum disk size 20 Gi")

	return WrapValidate(v)
}

type DiskList struct {
	Project  string `json:"project" yaml:"project"`
	Location string `json:"location" yaml:"location"`
}

func (m *DiskList) Valid() error {
	v := validator.New()

	v.Must(m.Project != "", "project required")

	return WrapValidate(v)
}

type DiskListResult struct {
	List []*DiskItem
}

type DiskItem struct {
	ID        int64
	ProjectID int64
	Location  string
	Name      string
	Size      int64
	Status    Status
	Action    string
	CreatedAt time.Time
	CreatedBy string
	SuccessAt time.Time
}

type DiskDelete struct {
	Project string
	Name    string
}

func (m *DiskDelete) Valid() error {
	m.Name = strings.TrimSpace(m.Name)

	v := validator.New()

	v.Must(ReValidName.MatchString(m.Name), "name invalid "+ReValidNameStr)
	if cnt := utf8.RuneCountInString(m.Name); cnt > MaxNameLength {
		return fmt.Errorf("name invalid")
	}
	v.Must(m.Project != "", "project required")

	return WrapValidate(v)
}
