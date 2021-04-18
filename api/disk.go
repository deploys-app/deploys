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
	Create(ctx context.Context, m *DiskCreate) (*Empty, error)
	Get(ctx context.Context, m *DiskGet) (*DiskItem, error)
	List(ctx context.Context, m *DiskList) (*DiskListResult, error)
	Update(ctx context.Context, m *DiskUpdate) (*Empty, error)
	Delete(ctx context.Context, m *DiskDelete) (*Empty, error)
}

type DiskCreate struct {
	Location string `json:"location" yaml:"location"`
	Project  string `json:"project" yaml:"project"`
	Name     string `json:"name" yaml:"name"`
	Size     int64  `json:"size" yaml:"size"`
}

func (m *DiskCreate) Valid() error {
	m.Name = strings.TrimSpace(m.Name)

	v := validator.New()

	v.Must(m.Location != "", "location required")
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

type DiskUpdate struct {
	Location string `json:"location" yaml:"location"`
	Project  string `json:"project" yaml:"project"`
	Name     string `json:"name" yaml:"name"`
	Size     int64  `json:"size" yaml:"size"`
}

func (m *DiskUpdate) Valid() error {
	m.Name = strings.TrimSpace(m.Name)

	v := validator.New()

	v.Must(m.Location != "", "location required")
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

type DiskGet struct {
	Location string `json:"location" yaml:"location"`
	Project  string `json:"project" yaml:"project"`
	Name     string `json:"name" yaml:"name"`
}

func (m *DiskGet) Valid() error {
	v := validator.New()

	v.Must(m.Location != "", "location required")
	v.Must(m.Project != "", "project required")
	v.Must(m.Name != "", "name required")

	return WrapValidate(v)
}

type DiskList struct {
	Location string `json:"location" yaml:"location"` // optional
	Project  string `json:"project" yaml:"project"`
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
	ID        int64     `json:"id"`
	ProjectID int64     `json:"projectId"`
	Location  string    `json:"location"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	Status    Status    `json:"status"`
	Action    string    `json:"action"`
	CreatedAt time.Time `json:"createdAt"`
	CreatedBy string    `json:"createdBy"`
	SuccessAt time.Time `json:"successAt"`
}

type DiskDelete struct {
	Location string `json:"location" yaml:"location"`
	Project  string `json:"project" yaml:"project"`
	Name     string `json:"name" yaml:"name"`
}

func (m *DiskDelete) Valid() error {
	m.Name = strings.TrimSpace(m.Name)

	v := validator.New()

	v.Must(m.Location != "", "location required")
	v.Must(m.Project != "", "project required")
	v.Must(ReValidName.MatchString(m.Name), "name invalid "+ReValidNameStr)
	if cnt := utf8.RuneCountInString(m.Name); cnt > MaxNameLength {
		return fmt.Errorf("name invalid")
	}

	return WrapValidate(v)
}
