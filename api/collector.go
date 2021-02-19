package api

import (
	"context"
)

type Collector interface {
	Location(ctx context.Context, m CollectorLocation) (*CollectorLocationResult, error)
	SetProjectUsage(ctx context.Context, m CollectorSetProjectUsage) (*Empty, error)
	SetDeploymentUsage(ctx context.Context, m CollectorSetDeploymentUsage) (*Empty, error)
}

type CollectorLocation struct {
	Location string `json:"location"`
}

type CollectorLocationResult struct {
	Projects []*CollectorProject `json:"projects"`
}

type CollectorProject struct {
	ID int64 `json:"id"`
}

type CollectorSetProjectUsage struct {
	Location  string                           `json:"location"`
	ProjectID int64                            `json:"projectId"`
	At        string                           `json:"at"`
	Resources []*CollectorProjectUsageResource `json:"resources"`
}

type CollectorProjectUsageResource struct {
	Name  string `json:"name"`
	Value string `json:"value"` // decimal
}

type CollectorSetDeploymentUsage struct {
	Location string                          `json:"location"`
	List     []*CollectorDeploymentUsageItem `json:"list"`
}

type CollectorDeploymentUsageItem struct {
	ProjectID      int64   `json:"projectId"`
	DeploymentName string  `json:"deploymentName"`
	Pod            string  `json:"pod"`
	Name           string  `json:"name"`
	Value          float64 `json:"value"`
	At             int64   `json:"at"`
}
