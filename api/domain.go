package api

import (
	"context"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/moonrhythm/validator"
)

type Domain interface {
	Create(ctx context.Context, m *DomainCreate) (*Empty, error)
	Get(ctx context.Context, m *DomainGet) (*DomainItem, error)
	List(ctx context.Context, m *DomainList) (*DomainListResult, error)
	Delete(ctx context.Context, m *DomainDelete) (*Empty, error)
	PurgeCache(ctx context.Context, m *DomainPurgeCache) (*Empty, error)
}

type DomainCreate struct {
	Project  string     `json:"project" yaml:"project"`
	Location string     `json:"location" yaml:"location"`
	Domain   string     `json:"domain" yaml:"domain"`
	Wildcard bool       `json:"wildcard" yaml:"wildcard"`
	Type     DomainType `json:"type" yaml:"type"` // deprecate
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
	Project string `json:"project" yaml:"project"`
	Domain  string `json:"domain" yaml:"domain"`
}

func (m *DomainGet) Valid() error {
	v := validator.New()

	v.Must(m.Project != "", "project required")
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
	Project      string             `json:"project" yaml:"project"`
	Location     string             `json:"location" yaml:"location"`
	Domain       string             `json:"domain" yaml:"domain"`
	Type         DomainType         `json:"type" yaml:"type"` // deprecate
	Wildcard     bool               `json:"wildcard" yaml:"wildcard"`
	Verification DomainVerification `json:"verification" yaml:"verification"`
	DNSConfig    DomainDNSConfig    `json:"dnsConfig" yaml:"dnsConfig"`
	Status       DomainStatus       `json:"status" yaml:"status"`
	CreatedAt    time.Time          `json:"createdAt" yaml:"createdAt"`
	CreatedBy    string             `json:"createdBy" yaml:"createdBy"`
}

type DomainVerification struct {
	Ownership DomainVerificationOwnership `json:"ownership"`
	SSL       DomainVerificationSSL       `json:"ssl"`
}

type DomainVerificationOwnership struct {
	Type   string   `json:"type"`
	Name   string   `json:"name"`
	Value  string   `json:"value"`
	Errors []string `json:"errors"`
}

type DomainVerificationSSL struct {
	Pending bool                          `json:"pending"`
	Records []DomainVerificationSSLRecord `json:"records"`
	Errors  []string                      `json:"errors"`
}

type DomainVerificationSSLRecord struct {
	TxtName  string `json:"txtName"`
	TxtValue string `json:"txtValue"`
}

type DomainDNSConfig struct {
	IPv4  []string `json:"ipv4" yaml:"ipv4"`
	IPv6  []string `json:"ipv6" yaml:"ipv6"`
	CName []string `json:"cname" yaml:"cname"`
}

type DomainDelete struct {
	Project string `json:"project" yaml:"project"`
	Domain  string `json:"domain" yaml:"domain"`
}

func (m *DomainDelete) Valid() error {
	v := validator.New()

	v.Must(m.Project != "", "project required")
	v.Must(govalidator.IsDNSName(m.Domain), "domain invalid")

	return WrapValidate(v)
}

type DomainPurgeCache struct {
	Project string `json:"project" yaml:"project"`
	Domain  string `json:"domain" yaml:"domain"`
	Prefix  string `json:"prefix" yaml:"prefix"`
}

func (m *DomainPurgeCache) Valid() error {
	v := validator.New()

	m.Domain = strings.TrimSpace(m.Domain)
	m.Prefix = strings.TrimSpace(m.Prefix)

	v.Must(m.Project != "", "project required")
	v.Must(govalidator.IsDNSName(m.Domain), "domain invalid")

	return WrapValidate(v)
}
