package api

import (
	"context"
	"fmt"
	"strconv"
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
	Items []*DiskItem `json:"items" yaml:"items"`
	List  []*DiskItem `json:"list" yaml:"list"`
}

func (m *DiskListResult) Table() [][]string {
	table := [][]string{
		{"NAME", "SIZE", "LOCATION", "AGE"},
	}
	for _, x := range m.Items {
		table = append(table, []string{
			x.Name,
			strconv.FormatInt(x.Size, 10) + "Gi",
			x.Location,
			age(x.CreatedAt),
		})
	}
	return table
}

type DiskItem struct {
	ID        int64     `json:"id" yaml:"id"`
	ProjectID int64     `json:"projectId" yaml:"projectId"`
	Location  string    `json:"location" yaml:"location"`
	Name      string    `json:"name" yaml:"name"`
	Size      int64     `json:"size" yaml:"size"`
	Status    Status    `json:"status" yaml:"status"`
	Action    string    `json:"action" yaml:"action"`
	CreatedAt time.Time `json:"createdAt" yaml:"createdAt"`
	CreatedBy string    `json:"createdBy" yaml:"createdBy"`
	SuccessAt time.Time `json:"successAt" yaml:"successAt"`
}

func (m *DiskItem) Table() [][]string {
	table := [][]string{
		{"NAME", "SIZE", "LOCATION", "AGE"},
		{
			m.Name,
			strconv.FormatInt(m.Size, 10) + "Gi",
			m.Location,
			age(m.CreatedAt),
		},
	}
	return table
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
