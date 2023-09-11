package api

import (
	"context"
)

type Collector interface {
	Location(ctx context.Context, m *CollectorLocation) (*CollectorLocationResult, error)
	SetProjectUsage(ctx context.Context, m *CollectorSetProjectUsage) (*Empty, error)
	SetDeploymentUsage(ctx context.Context, m *CollectorSetDeploymentUsage) (*Empty, error)
	SetDiskUsage(ctx context.Context, m *CollectorSetDiskUsage) (*Empty, error)
}

type CollectorLocation struct {
	Location string `json:"location" yaml:"location"`
}

type CollectorLocationResult struct {
	Projects []*CollectorProject `json:"projects" yaml:"projects"`
}

type CollectorProject struct {
	ID int64 `json:"id" yaml:"id"`
}

type CollectorSetProjectUsage struct {
	Location  string                           `json:"location" yaml:"location"`
	ProjectID int64                            `json:"projectId" yaml:"projectId"`
	At        string                           `json:"at" yaml:"at"`
	Resources []*CollectorProjectUsageResource `json:"resources" yaml:"resources"`
}

type CollectorProjectUsageResource struct {
	Name  string `json:"name" yaml:"name"`
	Value string `json:"value" yaml:"value"` // decimal
}

type CollectorSetDeploymentUsage struct {
	Location string                          `json:"location" yaml:"location"`
	List     []*CollectorDeploymentUsageItem `json:"list" yaml:"list"`
}

type CollectorDeploymentUsageItem struct {
	ProjectID      int64   `json:"projectId" yaml:"projectId"`
	DeploymentName string  `json:"deploymentName" yaml:"deploymentName"`
	Pod            string  `json:"pod" yaml:"pod"`
	Name           string  `json:"name" yaml:"name"`
	Value          float64 `json:"value" yaml:"value"`
	At             int64   `json:"at" yaml:"at"`
}

type CollectorSetDiskUsage struct {
	Location string                    `json:"location" yaml:"location"`
	List     []*CollectorDiskUsageItem `json:"list" yaml:"list"`
}

type CollectorDiskUsageItem struct {
	ProjectID int64   `json:"projectId" yaml:"projectId"`
	DiskName  string  `json:"diskName" yaml:"diskName"`
	Name      string  `json:"name" yaml:"name"`
	Value     float64 `json:"value" yaml:"value"`
	At        int64   `json:"at" yaml:"at"`
}
