package api

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/asaskevich/govalidator"
	"github.com/moonrhythm/validator"
)

type PullSecret interface {
	Create(ctx context.Context, m *PullSecretCreate) (*Empty, error)
	Get(ctx context.Context, m *PullSecretGet) (*PullSecretItem, error)
	List(ctx context.Context, m *PullSecretList) (*PullSecretListResult, error)
	Delete(ctx context.Context, m *PullSecretDelete) (*Empty, error)
}

type PullSecretCreate struct {
	Project  string         `json:"project" yaml:"project"`
	Location string         `json:"location" yaml:"location"`
	Name     string         `json:"name" yaml:"name"`
	Spec     PullSecretSpec `json:"spec" yaml:"spec"`
}

type PullSecretSpec struct {
	Server   string `json:"server" yaml:"server"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
}

func (m *PullSecretCreate) Valid() error {
	m.Name = strings.TrimSpace(m.Name)
	m.Spec.Server = strings.TrimSpace(m.Spec.Server)
	m.Spec.Username = strings.TrimSpace(m.Spec.Username)

	v := validator.New()

	v.Must(m.Project != "", "project required")
	v.Must(m.Location != "", "location required")
	v.Mustf(ReValidName.MatchString(m.Name), "name invalid %s", ReValidNameStr)
	{
		cnt := utf8.RuneCountInString(m.Name)
		v.Mustf(cnt >= MinNameLength && cnt <= MaxNameLength, "name must have length between %d-%d characters", MinNameLength, MaxNameLength)
	}
	v.Must(govalidator.IsURL(m.Spec.Server), "spec.server must be an url")
	v.Must(m.Spec.Server != "", "spec.server required")
	v.Must(m.Spec.Username != "", "spec.username required")
	v.Must(m.Spec.Password != "", "spec.password required")

	return WrapValidate(v)
}

type PullSecretDelete struct {
	Location string `json:"location" yaml:"location"`
	Project  string `json:"project" yaml:"project"`
	Name     string `json:"name" yaml:"name"`
}

func (m *PullSecretDelete) Valid() error {
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

type PullSecretList struct {
	Project  string `json:"project" yaml:"project"`
	Location string `json:"location" yaml:"location"`
}

func (m *PullSecretList) Valid() error {
	v := validator.New()

	v.Must(m.Project != "", "project required")

	return WrapValidate(v)
}

type PullSecretListResult struct {
	Project  string            `json:"project" yaml:"project"`
	Location string            `json:"location" yaml:"location"`
	Items    []*PullSecretItem `json:"items" yaml:"items"`
}

func (m *PullSecretListResult) Table() [][]string {
	table := [][]string{
		{"NAME", "STATUS", "LOCATION", "AGE"},
	}
	for _, x := range m.Items {
		table = append(table, []string{
			x.Name,
			x.Status.Text(),
			x.Location,
			age(x.CreatedAt),
		})
	}
	return table
}

type PullSecretItem struct {
	Name      string         `json:"name" yaml:"name"`
	Value     string         `json:"value" yaml:"value"`
	Spec      PullSecretSpec `json:"spec" yaml:"spec"`
	Location  string         `json:"location" yaml:"location"`
	Action    Action         `json:"action" yaml:"action"`
	Status    Status         `json:"status" yaml:"status"`
	CreatedAt time.Time      `json:"createdAt" yaml:"createdAt"`
	CreatedBy string         `json:"createdBy" yaml:"createdBy"`
}

func (m *PullSecretItem) Table() [][]string {
	table := [][]string{
		{"NAME", "STATUS", "LOCATION", "AGE"},
		{
			m.Name,
			m.Status.Text(),
			m.Location,
			age(m.CreatedAt),
		},
	}
	return table
}

type PullSecretGet struct {
	Location string `json:"location" yaml:"location"`
	Project  string `json:"project" yaml:"project"`
	Name     string `json:"name" yaml:"name"`
}

func (m *PullSecretGet) Valid() error {
	v := validator.New()

	v.Must(m.Location != "", "location required")
	v.Must(m.Project != "", "project required")
	v.Must(m.Name != "", "name required")

	return WrapValidate(v)
}
