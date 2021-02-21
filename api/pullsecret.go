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
	Project  string `json:"project" yaml:"project"`
	Location string `json:"location" yaml:"location"`
	Name     string `json:"name" yaml:"name"`
	Value    string `json:"value" yaml:"value"`
}

func (m *PullSecretCreate) Valid() error {
	m.Name = strings.TrimSpace(m.Name)
	m.Value = strings.TrimSpace(m.Value)

	v := validator.New()

	v.Must(m.Project != "", "project required")
	v.Must(m.Location != "", "location required")
	v.Mustf(ReValidName.MatchString(m.Name), "name invalid %s", ReValidNameStr)
	{
		cnt := utf8.RuneCountInString(m.Name)
		v.Mustf(cnt >= MinNameLength && cnt <= MaxNameLength, "name must have length between %d-%d characters", MinNameLength, MaxNameLength)
	}
	v.Must(m.Value != "", "value required")
	v.Must(govalidator.IsBase64(m.Value), "value must be base64")

	return WrapValidate(v)
}

type PullSecretDelete struct {
	Project string `json:"project" yaml:"project"`
	Name    string `json:"name" yaml:"name"`
}

func (m *PullSecretDelete) Valid() error {
	m.Name = strings.TrimSpace(m.Name)

	v := validator.New()

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
	Project     string            `json:"project" yaml:"project"`
	Location    string            `json:"location" yaml:"location"`
	PullSecrets []*PullSecretItem `json:"pullSecrets" yaml:"pullSecrets"`
}

func (m *PullSecretListResult) Table() [][]string {
	table := [][]string{
		{"NAME", "STATUS", "LOCATION", "AGE"},
	}
	for _, x := range m.PullSecrets {
		table = append(table, []string{
			x.Name,
			x.Status.String(),
			x.Location,
			age(x.CreatedAt),
		})
	}
	return table
}

type PullSecretItem struct {
	Name      string    `json:"name" yaml:"name"`
	Value     string    `json:"value" yaml:"value"`
	Location  string    `json:"location" yaml:"location"`
	Action    Action    `json:"action" yaml:"action"`
	Status    Status    `json:"status" yaml:"status"`
	CreatedAt time.Time `json:"createdAt" yaml:"createdAt"`
	CreatedBy string    `json:"createdBy" yaml:"createdBy"`
}

type PullSecretGet struct {
	Project string `json:"project" yaml:"project"`
	Name    string `json:"name" yaml:"name"`
}
