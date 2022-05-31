package api

import (
	"context"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/moonrhythm/validator"
)

type Domain interface {
	Create(ctx context.Context, m *DomainCreate) (*Empty, error)
	Get(ctx context.Context, m *DomainGet) (*DomainItem, error)
	List(ctx context.Context, m *DomainList) (*DomainListResult, error)
	Delete(ctx context.Context, m *DomainDelete) (*Empty, error)
}

type DomainCreate struct {
	Project  string `json:"project" yaml:"project"`
	Location string `json:"location" yaml:"location"`
	Domain   string `json:"domain" yaml:"domain"`
}

func (m *DomainCreate) Valid() error {
	v := validator.New()

	v.Must(m.Project != "", "project required")
	v.Must(m.Location != "", "location required")
	v.Must(govalidator.IsDNSName(m.Domain), "domain invalid")
	v.Must(!strings.HasSuffix(m.Domain, ".deploys.app"), "domain invalid")

	return WrapValidate(v)
}

type DomainGet struct {
	Project  string `json:"project" yaml:"project"`
	Location string `json:"location" yaml:"location"`
	Domain   string `json:"domain" yaml:"domain"`
}

func (m *DomainGet) Valid() error {
	v := validator.New()

	v.Must(m.Project != "", "project required")
	v.Must(m.Location != "", "location required")
	v.Must(govalidator.IsDNSName(m.Domain), "domain invalid")

	return WrapValidate(v)
}

type DomainList struct {
	Project  string `json:"project" yaml:"project"`
	Location string `json:"location" yaml:"location"`
}

func (m *DomainList) Valid() error {
	v := validator.New()

	v.Must(m.Project != "", "project required")

	return WrapValidate(v)
}

type DomainListResult struct {
	Items []*DomainItem `json:"items" yaml:"items"`
}

func (m *DomainListResult) Table() [][]string {
	table := [][]string{
		{"DOMAIN", "LOCATION"},
	}
	for _, x := range m.Items {
		table = append(table, []string{
			x.Domain,
			x.Location,
		})
	}
	return table
}

type DomainItem struct {
	Project  string `json:"project" yaml:"project"`
	Location string `json:"location" yaml:"location"`
	Domain   string `json:"domain" yaml:"domain"`
}

type DomainDelete struct {
	Project  string `json:"project" yaml:"project"`
	Location string `json:"location" yaml:"location"`
	Domain   string `json:"domain" yaml:"domain"`
}

func (m *DomainDelete) Valid() error {
	v := validator.New()

	v.Must(m.Project != "", "project required")
	v.Must(m.Location != "", "location required")
	v.Must(govalidator.IsDNSName(m.Domain), "domain invalid")

	return WrapValidate(v)
}
