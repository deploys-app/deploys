package api

import (
	"context"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dustin/go-humanize"
	"github.com/moonrhythm/validator"
)

type Project interface {
	Create(ctx context.Context, m *ProjectCreate) (*Empty, error)
	Get(ctx context.Context, m *ProjectGet) (*ProjectItem, error)
	List(ctx context.Context, m *Empty) (*ProjectListResult, error)
	Update(ctx context.Context, m *ProjectUpdate) (*Empty, error)
	Delete(ctx context.Context, m *ProjectDelete) (*Empty, error)
	Usage(ctx context.Context, m *ProjectUsage) (*ProjectUsageResult, error)
}

type ProjectCreate struct {
	SID            string `json:"sid" yaml:"sid"`
	Name           string `json:"name" yaml:"name"`
	BillingAccount string `json:"billingAccount" yaml:"billingAccount"`
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
		v.Must(cnt >= 6 && cnt <= 32, "sid must have length between 6-32 characters")
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
	Project string `json:"project" yaml:"project"`
}

type ProjectItem struct {
	ID             int64         `json:"id" yaml:"id"`
	Project        string        `json:"project" yaml:"project"`
	Name           string        `json:"name" yaml:"name"`
	BillingAccount string        `json:"billingAccount" yaml:"billingAccount"`
	Quota          ProjectQuota  `json:"quota" yaml:"quota"`
	Config         ProjectConfig `json:"config" yaml:"config"`
	CreatedAt      time.Time     `json:"createdAt" yaml:"createdAt"`
}

func (m *ProjectItem) Table() [][]string {
	return [][]string{
		{"PROJECT", "NAME", "AGE"},
		{
			m.Project,
			m.Name,
			age(m.CreatedAt),
		},
	}
}

type ProjectQuota struct {
	Deployments           int `json:"deployments" yaml:"deployments"`
	DeploymentMaxReplicas int `json:"deploymentMaxReplicas" yaml:"deploymentMaxReplicas"`
}

type ProjectConfig struct {
	DomainCloudflare      bool `json:"domainCloudflare" yaml:"domainCloudflare"`
	DomainAllowDisableCDN bool `json:"domainAllowDisableCdn" yaml:"domainAllowDisableCDN"`
	DomainWildcard        bool `json:"domainWildcard" yaml:"domainWildcard"`
}

type ProjectListResult struct {
	Items []*ProjectItem `json:"items" yaml:"items"`
}

func (m *ProjectListResult) Table() [][]string {
	table := [][]string{
		{"PROJECT", "NAME", "AGE"},
	}
	for _, x := range m.Items {
		table = append(table, []string{
			x.Project,
			x.Name,
			age(x.CreatedAt),
		})
	}
	return table
}

type ProjectDelete struct {
	Project string `json:"project" yaml:"project"`
}

type ProjectUsage struct {
	Project string `json:"project" yaml:"project"`
}

type ProjectUsageResult struct {
	CPUUsage float64 `json:"cpuUsage" yaml:"cpuUsage"`
	CPU      float64 `json:"cpu" yaml:"cpu"`
	Memory   float64 `json:"memory" yaml:"memory"`
	Egress   float64 `json:"egress" yaml:"egress"`
	Disk     float64 `json:"disk" yaml:"disk"`
	Replica  float64 `json:"replica" yaml:"replica"`
}

func (m *ProjectUsageResult) Table() [][]string {
	table := [][]string{
		{"RESOURCE", "USAGE"},
		{"CPUUsage", humanize.CommafWithDigits(m.CPUUsage, 2)},
		{"CPU", humanize.CommafWithDigits(m.CPU, 2)},
		{"Memory", humanize.CommafWithDigits(m.Memory, 2)},
		{"Egress", humanize.CommafWithDigits(m.Egress, 2)},
		{"Disk", humanize.CommafWithDigits(m.Disk, 2)},
		{"Replica", humanize.CommafWithDigits(m.Replica, 2)},
	}
	return table
}
