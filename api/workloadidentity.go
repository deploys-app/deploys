package api

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/asaskevich/govalidator"
	"github.com/moonrhythm/validator"
)

type WorkloadIdentity interface {
	Create(ctx context.Context, m WorkloadIdentityCreate) (*Empty, error)
	Get(ctx context.Context, m WorkloadIdentityGet) (*WorkloadIdentityGetResult, error)
	List(ctx context.Context, m WorkloadIdentityList) (*WorkloadIdentityListResult, error)
	Delete(ctx context.Context, m WorkloadIdentityDelete) (*Empty, error)
}

type WorkloadIdentityCreate struct {
	ProjectID int64  `json:"projectId" yaml:"projectId"`
	Location  string `json:"location" yaml:"location"`
	Name      string `json:"name" yaml:"name"`
	GSA       string `json:"gsa" yaml:"gsa"`
}

func (m *WorkloadIdentityCreate) Valid() error {
	m.Name = strings.TrimSpace(m.Name)
	m.GSA = strings.TrimSpace(m.GSA)

	v := validator.New()
	v.Must(ReValidName.MatchString(m.Name), "name invalid "+ReValidNameStr)
	{
		cnt := utf8.RuneCountInString(m.Name)
		v.Mustf(cnt >= MinNameLength && cnt <= MaxNameLength, "name must have length between %d-%d characters", MinNameLength, MaxNameLength)
	}
	v.Must(m.ProjectID > 0, "project id required")
	v.Must(m.GSA == "" || govalidator.IsEmail(m.GSA), "gsa must be an email")
	v.Must(strings.HasSuffix(m.GSA, ".iam.gserviceaccount.com"), "gsa must end with '.iam.gserviceaccount.com'")

	return WrapValidate(v)
}

type WorkloadIdentityGet struct {
	ProjectID int64  `json:"projectId" yaml:"projectId"`
	Name      string `json:"name" yaml:"name"`
}

func (m *WorkloadIdentityGet) Valid() error {
	m.Name = strings.TrimSpace(m.Name)

	v := validator.New()
	v.Must(ReValidName.MatchString(m.Name), "name invalid "+ReValidNameStr)
	{
		cnt := utf8.RuneCountInString(m.Name)
		v.Mustf(cnt >= MinNameLength && cnt <= MaxNameLength, "name must have length between %d-%d characters", MinNameLength, MaxNameLength)
	}
	v.Must(m.ProjectID > 0, "project id required")

	return WrapValidate(v)
}

type WorkloadIdentityGetResult struct {
	ID         int64     `json:"id" yaml:"id"`
	ProjectID  int64     `json:"projectId" yaml:"projectId"`
	LocationID string    `json:"location" yaml:"location"`
	Name       string    `json:"name" yaml:"name"`
	GSA        string    `json:"gsa" yaml:"gsa"`
	Status     Status    `json:"status" yaml:"status"`
	Action     Action    `json:"action" yaml:"action"`
	CreatedAt  time.Time `json:"createdAt" yaml:"createdAt"`
	CreatedBy  string    `json:"createdBy" yaml:"createdBy"`
}

func (m *WorkloadIdentityGetResult) ResourceID() string {
	return fmt.Sprintf("%s-%d", m.Name, m.ProjectID)
}

type WorkloadIdentityDelete struct {
	ProjectID int64  `json:"projectId" yaml:"projectId"`
	Name      string `json:"name" yaml:"name"`
}

func (m *WorkloadIdentityDelete) Valid() error {
	m.Name = strings.TrimSpace(m.Name)

	v := validator.New()
	v.Must(ReValidName.MatchString(m.Name), "name invalid "+ReValidNameStr)
	{
		cnt := utf8.RuneCountInString(m.Name)
		v.Mustf(cnt >= MinNameLength && cnt <= MaxNameLength, "name must have length between %d-%d characters", MinNameLength, MaxNameLength)
	}
	v.Must(m.ProjectID > 0, "project id required")

	return WrapValidate(v)
}

type WorkloadIdentityList struct {
	ProjectID int64  `json:"projectId" yaml:"projectId"`
	Location  string `json:"location" yaml:"location"`
}

func (m *WorkloadIdentityList) Valid() error {
	v := validator.New()

	v.Must(m.ProjectID > 0, "project id required")

	return WrapValidate(v)
}

type WorkloadIdentityItem struct {
	ID         int64     `json:"id" yaml:"id"`
	ProjectID  int64     `json:"projectId" yaml:"projectId"`
	LocationID string    `json:"locationId" yaml:"locationId"`
	Name       string    `json:"name" yaml:"name"`
	GSA        string    `json:"gsa" yaml:"gsa"`
	Status     Status    `json:"status" yaml:"status"`
	Action     Action    `json:"action" yaml:"action"`
	CreatedAt  time.Time `json:"createdAt" yaml:"createdAt"`
	CreatedBy  string    `json:"createdBy" yaml:"createdBy"`
}

func (m *WorkloadIdentityItem) ResourceID() string {
	return fmt.Sprintf("%s-%d", m.Name, m.ProjectID)
}

type WorkloadIdentityListResult struct {
	List []*WorkloadIdentityItem
}
