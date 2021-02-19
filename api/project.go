package api

import (
	"context"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/moonrhythm/validator"
)

type Project interface {
	Create(ctx context.Context, m ProjectCreate) (*Empty, error)
	Get(ctx context.Context, m ProjectGet) (*ProjectGetResult, error)
	List(ctx context.Context, m Empty) (*ProjectListResult, error)
	Update(ctx context.Context, m ProjectUpdate) (*Empty, error)
	Usage(ctx context.Context, m ProjectUsage) (*ProjectUsageResult, error)
}

type ProjectCreate struct {
	SID            string
	Name           string
	BillingAccount string
}

var (
	ReValidSIDStr = `^[a-z][a-z0-9\-]*[^\-]$`
	ReValidSID    = regexp.MustCompile(ReValidSIDStr)
)

func (m *ProjectCreate) Valid() error {
	m.SID = strings.TrimSpace(m.SID)
	m.Name = strings.TrimSpace(m.Name)

	v := validator.New()

	if v.Must(m.SID != "", "sid required") {
		v.Mustf(ReValidSID.MatchString(m.SID), "sid invalid %s", ReValidSIDStr)
		cnt := utf8.RuneCountInString(m.SID)
		v.Must(cnt >= 6 && cnt <= 20, "sid must have length between 6-20 characters")
	}

	v.Must(utf8.ValidString(m.Name), "name invalid")
	cnt := utf8.RuneCountInString(m.Name)
	v.Must(cnt >= 4 && cnt <= 64, "name must have length between 4-64 characters")

	v.Must(m.BillingAccount != "", "billing account required")

	return WrapValidate(v)
}

type ProjectUpdate struct {
	Project        string  `json:"project" yaml:"project"`
	Name           *string `json:"name" yaml:"name"`
	BillingAccount *string `json:"billingAccount" yaml:"billingAccount"`
}

func (m *ProjectUpdate) Valid() error {
	m.Project = strings.TrimSpace(m.Project)

	v := validator.New()

	v.Must(m.Project != "", "project rquired")

	if m.Name != nil {
		*m.Name = strings.TrimSpace(*m.Name)
		v.Must(utf8.ValidString(*m.Name), "name invalid")
		cnt := utf8.RuneCountInString(*m.Name)
		v.Must(cnt >= 4 && cnt <= 64, "name must have length between 4-64 characters")
	}

	if m.BillingAccount != nil {
		v.Must(*m.BillingAccount != "", "billing account invalid")
	}

	return WrapValidate(v)
}

type ProjectGet struct {
	Project string `json:"project"`
}

type ProjectGetResult struct {
	ID             int64     `json:"id" yaml:"id"`
	Project        string    `json:"project" yaml:"project"`
	Name           string    `json:"name" yaml:"name"`
	BillingAccount string    `json:"billingAccount" yaml:"billingAccount"`
	CreatedAt      time.Time `json:"createdAt" yaml:"createdAt"`
}

func (m ProjectGetResult) Table() [][]string {
	return [][]string{
		{"PROJECT", "NAME", "AGE"},
		{
			m.Project,
			m.Name,
			age(m.CreatedAt),
		},
	}
}

type ProjectListResult struct {
	Projects []*ProjectGetResult `json:"projects"`
}

func (m ProjectListResult) Table() [][]string {
	table := [][]string{
		{"PROJECT", "NAME", "AGE"},
	}
	for _, x := range m.Projects {
		table = append(table, []string{
			x.Project,
			x.Name,
			age(x.CreatedAt),
		})
	}
	return table
}

type ProjectUsage struct {
	Project string `json:"project"`
}

type ProjectUsageResult struct {
	CPUUsage float64 `json:"cpuUsage"`
	CPU      float64 `json:"cpu"`
	Memory   float64 `json:"memory"`
	Egress   float64 `json:"egress"`
	Disk     float64 `json:"disk"`
	Replica  float64 `json:"replica"`
}