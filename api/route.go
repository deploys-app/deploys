package api

import (
	"context"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/moonrhythm/validator"
)

var routeTargetPrefix = []string{
	"deployment://",
	"redirect://",
	"ipfs://",
	"ipns://",
	"dnslink://",
}

func RouteTargetPrefix() []string {
	xs := make([]string, len(routeTargetPrefix))
	copy(xs, routeTargetPrefix)
	return xs
}

type Route interface {
	Create(ctx context.Context, m *RouteCreate) (*Empty, error)
	CreateV2(ctx context.Context, m *RouteCreateV2) (*Empty, error)
	Get(ctx context.Context, m *RouteGet) (*RouteItem, error)
	List(ctx context.Context, m *RouteList) (*RouteListResult, error)
	ListV2(ctx context.Context, m *RouteList) (*RouteListV2Result, error)
	Delete(ctx context.Context, m *RouteDelete) (*Empty, error)
}

type RouteCreate struct {
	Project    string `json:"project" yaml:"project"`
	Location   string `json:"location" yaml:"location"`
	Domain     string `json:"domain" yaml:"domain"`
	Path       string `json:"path" yaml:"path"`
	Deployment string `json:"deployment" yaml:"deployment"`
}

func (m *RouteCreate) Valid() error {
	v := validator.New()

	v.Must(m.Project != "", "project required")
	v.Must(m.Location != "", "location required")
	v.Must(govalidator.IsDNSName(m.Domain), "domain invalid")
	v.Must(!strings.HasSuffix(m.Domain, ".deploys.app"), "domain invalid")
	if m.Path != "" {
		v.Must(strings.HasPrefix(m.Path, "/"), "path must start with /")
	}

	return WrapValidate(v)
}

type RouteCreateV2 struct {
	Project  string `json:"project" yaml:"project"`
	Location string `json:"location" yaml:"location"`
	Domain   string `json:"domain" yaml:"domain"`
	Path     string `json:"path" yaml:"path"`
	Target   string `json:"target" yaml:"target"`
}

func (m *RouteCreateV2) Valid() error {
	v := validator.New()

	v.Must(m.Project != "", "project required")
	v.Must(m.Location != "", "location required")
	v.Must(govalidator.IsDNSName(m.Domain), "domain invalid")
	v.Must(!strings.HasSuffix(m.Domain, ".deploys.app"), "domain invalid")
	if m.Path != "" {
		v.Must(strings.HasPrefix(m.Path, "/"), "path must start with /")
	}
	v.Must(validRouteTarget(m.Target), "target invalid")

	return WrapValidate(v)
}

type RouteGet struct {
	Project  string `json:"project" yaml:"project"`
	Location string `json:"location" yaml:"location"`
	Domain   string `json:"domain" yaml:"domain"`
	Path     string `json:"path" yaml:"path"`
}

func (m *RouteGet) Valid() error {
	v := validator.New()

	v.Must(m.Project != "", "project required")
	v.Must(m.Location != "", "location required")
	v.Must(govalidator.IsDNSName(m.Domain), "domain invalid")
	if m.Path != "" {
		v.Must(strings.HasPrefix(m.Path, "/"), "path must start with /")
	}

	return WrapValidate(v)
}

type RouteList struct {
	Project  string `json:"project" yaml:"project"`
	Location string `json:"location" yaml:"location"`
}

func (m *RouteList) Valid() error {
	v := validator.New()

	v.Must(m.Project != "", "project required")

	return WrapValidate(v)
}

type RouteListV2Result []*RouteItemV2

func (m RouteListV2Result) Table() [][]string {
	table := [][]string{
		{"DOMAIN", "PATH", "TARGET", "LOCATION"},
	}
	for _, x := range m {
		table = append(table, []string{
			x.Domain,
			x.Path,
			x.Target,
			x.Location,
		})
	}
	return table
}

type RouteItemV2 struct {
	Location string `json:"location" yaml:"location"`
	Domain   string `json:"domain" yaml:"domain"`
	Path     string `json:"path" yaml:"path"`
	Target   string `json:"target" yaml:"target"`
}

type RouteListResult struct {
	Items []*RouteItem `json:"items" yaml:"items"`
}

func (m *RouteListResult) Table() [][]string {
	table := [][]string{
		{"DOMAIN", "PATH", "DEPLOYMENT", "LOCATION"},
	}
	for _, x := range m.Items {
		table = append(table, []string{
			x.Domain,
			x.Path,
			x.Deployment,
			x.Location,
		})
	}
	return table
}

type RouteItem struct {
	Location   string `json:"location" yaml:"location"`
	Domain     string `json:"domain" yaml:"domain"`
	Path       string `json:"path" yaml:"path"`
	Deployment string `json:"deployment" yaml:"deployment"`
}

func (m *RouteItem) Table() [][]string {
	table := [][]string{
		{"DOMAIN", "PATH", "DEPLOYMENT", "LOCATION"},
		{
			m.Domain,
			m.Path,
			m.Deployment,
			m.Location,
		},
	}
	return table
}

type RouteDelete struct {
	Project  string `json:"project" yaml:"project"`
	Location string `json:"location" yaml:"location"`
	Domain   string `json:"domain" yaml:"domain"`
	Path     string `json:"path" yaml:"path"`
}

func (m *RouteDelete) Valid() error {
	v := validator.New()

	v.Must(m.Project != "", "project required")
	v.Must(m.Location != "", "location required")
	v.Must(govalidator.IsDNSName(m.Domain), "domain invalid")
	if m.Path != "" {
		v.Must(strings.HasPrefix(m.Path, "/"), "path must start with /")
	}

	return WrapValidate(v)
}
