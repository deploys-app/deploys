package api

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/asaskevich/govalidator"
	"github.com/moonrhythm/validator"
)

type WorkloadIdentity interface {
	Create(ctx context.Context, m *WorkloadIdentityCreate) (*Empty, error)
	Get(ctx context.Context, m *WorkloadIdentityGet) (*WorkloadIdentityItem, error)
	List(ctx context.Context, m *WorkloadIdentityList) (*WorkloadIdentityListResult, error)
	Delete(ctx context.Context, m *WorkloadIdentityDelete) (*Empty, error)
}

type WorkloadIdentityCreate struct {
	Location string `json:"location" yaml:"location"`
	Project  string `json:"project" yaml:"project"`
	Name     string `json:"name" yaml:"name"`
	GSA      string `json:"gsa" yaml:"gsa"`
}

func (m *WorkloadIdentityCreate) Valid() error {
	m.Name = strings.TrimSpace(m.Name)
	m.GSA = strings.TrimSpace(m.GSA)

	v := validator.New()
	v.Must(m.Location != "", "location required")
	v.Must(m.Project != "", "project required")
	v.Must(ReValidName.MatchString(m.Name), "name invalid "+ReValidNameStr)
	{
		cnt := utf8.RuneCountInString(m.Name)
		v.Mustf(cnt >= MinNameLength && cnt <= MaxNameLength, "name must have length between %d-%d characters", MinNameLength, MaxNameLength)
	}
	v.Must(m.GSA == "" || govalidator.IsEmail(m.GSA), "gsa must be an email")
	v.Must(strings.HasSuffix(m.GSA, ".iam.gserviceaccount.com"), "gsa must end with '.iam.gserviceaccount.com'")

	return WrapValidate(v)
}

type WorkloadIdentityGet struct {
	Location string `json:"location" yaml:"location"`
	Project  string `json:"project" yaml:"project"`
	Name     string `json:"name" yaml:"name"`
}

func (m *WorkloadIdentityGet) Valid() error {
	m.Name = strings.TrimSpace(m.Name)

	v := validator.New()
	v.Must(m.Location != "", "location required")
	v.Must(m.Project != "", "project required")
	v.Must(ReValidName.MatchString(m.Name), "name invalid "+ReValidNameStr)
	{
		cnt := utf8.RuneCountInString(m.Name)
		v.Mustf(cnt >= MinNameLength && cnt <= MaxNameLength, "name must have length between %d-%d characters", MinNameLength, MaxNameLength)
	}

	return WrapValidate(v)
}

type WorkloadIdentityDelete struct {
	Location string `json:"location" yaml:"location"`
	Project  string `json:"project" yaml:"project"`
	Name     string `json:"name" yaml:"name"`
}

func (m *WorkloadIdentityDelete) Valid() error {
	m.Name = strings.TrimSpace(m.Name)

	v := validator.New()
	v.Must(m.Location != "", "location required")
	v.Must(m.Project != "", "project required")
	v.Must(ReValidName.MatchString(m.Name), "name invalid "+ReValidNameStr)
	{
		cnt := utf8.RuneCountInString(m.Name)
		v.Mustf(cnt >= MinNameLength && cnt <= MaxNameLength, "name must have length between %d-%d characters", MinNameLength, MaxNameLength)
	}

	return WrapValidate(v)
}

type WorkloadIdentityList struct {
	Project  string `json:"project" yaml:"project"`
	Location string `json:"location" yaml:"location"`
}

func (m *WorkloadIdentityList) Valid() error {
	v := validator.New()

	v.Must(m.Project != "", "project required")

	return WrapValidate(v)
}

type WorkloadIdentityItem struct {
	Project   string    `json:"project" yaml:"project"`
	Location  string    `json:"location" yaml:"location"`
	Name      string    `json:"name" yaml:"name"`
	GSA       string    `json:"gsa" yaml:"gsa"`
	Status    Status    `json:"status" yaml:"status"`
	Action    Action    `json:"action" yaml:"action"`
	CreatedAt time.Time `json:"createdAt" yaml:"createdAt"`
	CreatedBy string    `json:"createdBy" yaml:"createdBy"`
}

func (m *WorkloadIdentityItem) Table() [][]string {
	table := [][]string{
		{"NAME", "GSA", "LOCATION", "AGE"},
		{
			m.Name,
			m.GSA,
			m.Location,
			age(m.CreatedAt),
		},
	}
	return table
}

type WorkloadIdentityListResult struct {
	Items []*WorkloadIdentityItem `json:"items" yaml:"items"`
}

func (m *WorkloadIdentityListResult) Table() [][]string {
	table := [][]string{
		{"NAME", "GSA", "LOCATION", "AGE"},
	}
	for _, x := range m.Items {
		table = append(table, []string{
			x.Name,
			x.GSA,
			x.Location,
			age(x.CreatedAt),
		})
	}
	return table
}
