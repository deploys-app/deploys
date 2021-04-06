package api

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/asaskevich/govalidator"
	"github.com/moonrhythm/validator"
)

type Billing interface {
	Create(ctx context.Context, m *BillingCreate) (*BillingCreateResult, error)
	List(ctx context.Context, m *Empty) (*BillingListResult, error)
	Delete(ctx context.Context, m *BillingDelete) (*Empty, error)
	Get(ctx context.Context, m *BillingGet) (*BillingItem, error)
	Update(ctx context.Context, m *BillingUpdate) (*Empty, error)
	Report(ctx context.Context, m *BillingReport) (*BillingReportResult, error)
	SKUs(ctx context.Context, m *Empty) (*BillingSKUs, error)
	Project(ctx context.Context, m *BillingProject) (*BillingProjectResult, error)
}

type BillingCreate struct {
	Name       string `json:"name" yaml:"name"`
	TaxID      string `json:"taxId" yaml:"taxId"`
	TaxName    string `json:"taxName" yaml:"taxName"`
	TaxAddress string `json:"taxAddress" yaml:"taxAddress"`
}

func (m *BillingCreate) Valid() error {
	m.Name = strings.TrimSpace(m.Name)
	m.TaxID = strings.TrimSpace(m.TaxID)
	m.TaxName = strings.TrimSpace(m.TaxName)
	m.TaxAddress = strings.TrimSpace(m.TaxAddress)

	v := validator.New()

	if ok := v.Must(m.Name != "", "name required"); ok {
		cnt := utf8.RuneCountInString(m.Name)
		v.Mustf(cnt >= MinNameLength && cnt <= MaxNameLength, "name must have length between %d-%d characters", MinNameLength, MaxNameLength)
	}
	v.Must(m.TaxID != "", "tax id required")
	v.Must(utf8.RuneCountInString(m.TaxID) < 100, "tax id too long")
	v.Must(m.TaxName != "", "tax name required")
	v.Must(utf8.RuneCountInString(m.TaxName) < 200, "tax name too long")
	v.Must(m.TaxAddress != "", "tax address required")
	v.Must(utf8.RuneCountInString(m.TaxAddress) < 500, "tax address too long")

	return WrapValidate(v)
}

type BillingCreateResult struct {
	ID string `json:"id" yaml:"id"`
}

type BillingListResult struct {
	Billings []*BillingItem `json:"billings" yaml:"billings"`
}

type BillingDelete struct {
	ID string `json:"id" yaml:"id"`
}

func (m *BillingDelete) Valid() error {
	v := validator.New()

	v.Must(m.ID != "", "id required")

	return WrapValidate(v)
}

type BillingGet struct {
	ID string `json:"id" yaml:"id"`
}

func (m *BillingGet) Valid() error {
	v := validator.New()

	v.Must(m.ID != "", "id required")
	v.Must(govalidator.IsNumeric(m.ID), "id must contain only number")

	return WrapValidate(v)
}

type BillingItem struct {
	ID         string `json:"id" yaml:"id"`
	Name       string `json:"name" yaml:"name"`
	TaxID      string `json:"taxId" yaml:"taxId"`
	TaxName    string `json:"taxName" yaml:"taxName"`
	TaxAddress string `json:"taxAddress" yaml:"taxAddress"`
}

type BillingUpdate struct {
	ID         string `json:"id" yaml:"id"`
	Name       string `json:"name" yaml:"name"`
	TaxID      string `json:"taxId" yaml:"taxId"`
	TaxName    string `json:"taxName" yaml:"taxName"`
	TaxAddress string `json:"taxAddress" yaml:"taxAddress"`
}

func (m *BillingUpdate) Valid() error {
	m.Name = strings.TrimSpace(m.Name)
	m.TaxID = strings.TrimSpace(m.TaxID)
	m.TaxName = strings.TrimSpace(m.TaxName)
	m.TaxAddress = strings.TrimSpace(m.TaxAddress)

	v := validator.New()

	if ok := v.Must(m.Name != "", "name required"); ok {
		cnt := utf8.RuneCountInString(m.Name)
		v.Mustf(cnt >= MinNameLength && cnt <= MaxNameLength, "name must have length between %d-%d characters", MinNameLength, MaxNameLength)
	}
	v.Must(m.TaxID != "", "tax id required")
	v.Must(utf8.RuneCountInString(m.TaxID) < 100, "tax id too long")
	v.Must(m.TaxName != "", "tax name required")
	v.Must(utf8.RuneCountInString(m.TaxName) < 200, "tax name too long")
	v.Must(m.TaxAddress != "", "tax address required")
	v.Must(utf8.RuneCountInString(m.TaxAddress) < 500, "tax address too long")

	return WrapValidate(v)
}

type BillingReport struct {
	ID          string   `json:"id" yaml:"id"`
	Range       string   `json:"range" yaml:"range"`
	ProjectSIDs []string `json:"projectSids" yaml:"projectSids"`
}

type BillingReportListItem struct {
	ProjectSID   string  `json:"projectSid" yaml:"projectSid"`
	Name         string  `json:"name" yaml:"name"`
	UsageValue   float64 `json:"usageValue" yaml:"usageValue"`
	BillingValue float64 `json:"billingValue" yaml:"billingValue"`
}

type BillingReportChartSeries struct {
	Name string    `json:"name" yaml:"name"`
	Data []float64 `json:"data" yaml:"data"`
}

type BillingReportChart struct {
	Categories []string                    `json:"categories" yaml:"categories"`
	Series     []*BillingReportChartSeries `json:"series" yaml:"series"`
}

type ReportProjectListItem struct {
	SID  string `json:"sid" yaml:"sid"`
	Name string `json:"name" yaml:"name"`
}

type BillingReportResult struct {
	Range       string                   `json:"range" yaml:"range"`
	List        []*BillingReportListItem `json:"list" yaml:"list"`
	Chart       *BillingReportChart      `json:"chart" yaml:"chart"`
	ProjectList []*ReportProjectListItem `json:"projectList" yaml:"projectList"`
	ProjectSIDs []string                 `json:"projectSids" yaml:"projectSids"`
}

type BillingSKUs struct {
	CPUUsage float64 `json:"cpuUsage"`
	CPU      float64 `json:"cpu"`
	Memory   float64 `json:"memory"`
	Egress   float64 `json:"egress"`
	Disk     float64 `json:"disk"`
}

type BillingProject struct {
	Project string `json:"project"`
}

type BillingProjectResult struct {
	Price float64 `json:"price"`
}
