package api

import (
	"context"
	"strings"
	"time"

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
	Location   string    `json:"location" yaml:"location"`
	Domain     string    `json:"domain" yaml:"domain"`
	Path       string    `json:"path" yaml:"path"`
	Target     string    `json:"target" yaml:"target"`
	Deployment string    `json:"deployment" yaml:"deployment"`
	CreatedAt  time.Time `json:"createdAt" yaml:"createdAt"`
	CreatedBy  string    `json:"createdBy" yaml:"createdBy"`
}

func (m *RouteItem) Table() [][]string {
	table := [][]string{
		{"DOMAIN", "PATH", "TARGET", "LOCATION"},
		{
			m.Domain,
			m.Path,
			m.Target,
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
