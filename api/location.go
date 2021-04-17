package api

import (
	"context"
	"time"

	"github.com/moonrhythm/validator"
)

type Location interface {
	List(ctx context.Context, _ *LocationList) (*LocationListResult, error)
	Get(ctx context.Context, m *LocationGet) (*LocationItem, error)
}

type LocationList struct {
	Project string `json:"project"` // optional
}

type LocationListResult struct {
	Locations []*LocationItem `json:"locations"`
}

func (m *LocationListResult) Table() [][]string {
	table := [][]string{
		{"ID", "DOMAIN SUFFIX", "ENDPOINT", "CNAME"},
	}
	for _, x := range m.Locations {
		table = append(table, []string{
			x.ID,
			x.DomainSuffix,
			x.Endpoint,
			x.CName,
		})
	}
	return table
}

type LocationItem struct {
	ID                string           `json:"id"`
	DomainSuffix      string           `json:"domainSuffix"`
	Endpoint          string           `json:"endpoint"`
	CName             string           `json:"cname"`
	FreeTier          bool             `json:"freeTier"`
	CPUAllocatable    []string         `json:"cpuAllocatable"`
	MemoryAllocatable []string         `json:"memoryAllocatable"`
	Features          LocationFeatures `json:"features"`
	CreatedAt         time.Time        `json:"createdAt"`
}

type LocationFeatures struct {
	WorkloadIdentity bool      `json:"workloadIdentity,omitempty"`
	Disk             *struct{} `json:"disk,omitempty"`
}

type LocationGet struct {
	ID string `json:"id"`
}

func (m *LocationGet) Valid() error {
	v := validator.New()
	v.Must(m.ID != "", "id required")
	return WrapValidate(v)
}
