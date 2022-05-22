package api

import (
	"context"
)

type Deployer interface {
	GetLocation(ctx context.Context, m *Empty) (*LocationItem, error)
	IsDomainActive(ctx context.Context, m *DeployerIsDomainActive) (bool, error)
	GetCommands(ctx context.Context, m *Empty) (*GetCommandsResult, error)
	SetResults(ctx context.Context, m *DeployerSetResult) (*Empty, error)
}

type DeployerIsDomainActive struct {
	Domain string `json:"domain"`
}

type GetCommandsResult []*DeployerCommandItem

type DeployerCommandItem struct {
	PullSecretCreate       *DeployerCommandPullSecretCreate       `json:"pullSecretCreate,omitempty"`
	PullSecretDelete       *DeployerCommandMetadata               `json:"pullSecretDelete,omitempty"`
	WorkloadIdentityCreate *DeployerCommandWorkloadIdentityCreate `json:"workloadIdentityCreate,omitempty"`
	WorkloadIdentityDelete *DeployerCommandMetadata               `json:"workloadIdentityDelete,omitempty"`
	DiskCreate             *DeployerCommandDiskCreate             `json:"diskCreate,omitempty"`
	DiskDelete             *DeployerCommandMetadata               `json:"diskDelete,omitempty"`
	DeploymentDeploy       *DeployerCommandDeploymentDeploy       `json:"deploymentDeploy,omitempty"`
	DeploymentDelete       *DeployerCommandDeploymentMetadata     `json:"deploymentDelete,omitempty"`
	DeploymentPause        *DeployerCommandDeploymentMetadata     `json:"deploymentPause,omitempty"`
	DeploymentCleanup      *DeployerCommandDeploymentMetadata     `json:"deploymentCleanup,omitempty"`
	RouteCreate            *DeployerCommandRouteCreate            `json:"routeCreate,omitempty"`
	RouteDelete            *DeployerCommandRouteDelete            `json:"routeDelete,omitempty"`
}

type DeployerCommandMetadata struct {
	ID        int64  `json:"id"`
	ProjectID int64  `json:"projectId"`
	Name      string `json:"name"`
}

type DeployerCommandPullSecretCreate struct {
	ID        int64  `json:"id"`
	ProjectID int64  `json:"projectId"`
	Name      string `json:"name"`
	Value     string `json:"value"`
}

type DeployerCommandWorkloadIdentityCreate struct {
	ID        int64  `json:"id"`
	ProjectID int64  `json:"projectId"`
	Name      string `json:"name"`
	GSA       string `json:"gsa"`
}

type DeployerCommandDiskCreate struct {
	ID        int64  `json:"id"`
	ProjectID int64  `json:"projectId"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
}

type DeployerCommandDeploymentMetadata struct {
	ID        int64          `json:"id"`
	ProjectID int64          `json:"projectId"`
	Name      string         `json:"name"`
	Revision  int64          `json:"revision"`
	Type      DeploymentType `json:"type"`
}

type DeployerCommandDeploymentDeploy struct {
	ID            int64                                        `json:"id"`
	ProjectID     int64                                        `json:"projectId"`
	Name          string                                       `json:"name"`
	Revision      int64                                        `json:"revision"`
	Type          DeploymentType                               `json:"type"`
	BillingConfig DeployerCommandDeploymentDeployBillingConfig `json:"config"`
	Spec          DeployerCommandDeploymentDeploySpec          `json:"spec"`
}

type DeployerCommandDeploymentDeployBillingConfig struct {
	Pool      string `json:"pool"`
	SharePool bool   `json:"sharePool"`
}

type DeployerCommandDeploymentDeploySpec struct {
	Image                string             `json:"image"`
	Env                  map[string]string  `json:"env"`
	Command              []string           `json:"command"`
	Args                 []string           `json:"args"`
	WorkloadIdentityName string             `json:"workloadIdentityName"`
	MinReplicas          int                `json:"minReplicas"`
	MaxReplicas          int                `json:"maxReplicas"`
	Port                 int                `json:"port"`
	Protocol             DeploymentProtocol `json:"protocol"`
	Schedule             string             `json:"schedule"`
	Annotations          map[string]string  `json:"annotations"`
	CPU                  string             `json:"cpu"`
	CPULimit             string             `json:"cpuLimit"`
	Memory               string             `json:"memory"`
	PullSecretName       string             `json:"pullSecretName"`
	DiskName             string             `json:"diskName"`
	DiskMountPath        string             `json:"diskMountPath"`
	DiskSubPath          string             `json:"diskSubPath"`
	MountData            map[string]string  `json:"mountData"` // file path => data
}

type DeployerCommandRouteCreate struct {
	ID        int64  `json:"id"`
	ProjectID int64  `json:"projectId"`
	Domain    string `json:"domain"`
	Path      string `json:"path"`
	Target    string `json:"target"`
}

type DeployerCommandRouteDelete struct {
	ID        int64  `json:"id"`
	ProjectID int64  `json:"projectId"`
	Domain    string `json:"domain"`
}

type DeployerSetResult []*DeployerSetResultItem

type DeployerSetResultItem struct {
	PullSecretCreate       *DeployerSetResultItemGeneral    `json:"pullSecretCreate,omitempty"`
	PullSecretDelete       *DeployerSetResultItemGeneral    `json:"pullSecretDelete,omitempty"`
	WorkloadIdentityCreate *DeployerSetResultItemGeneral    `json:"workloadIdentityCreate,omitempty"`
	WorkloadIdentityDelete *DeployerSetResultItemGeneral    `json:"workloadIdentityDelete,omitempty"`
	DiskCreate             *DeployerSetResultItemGeneral    `json:"diskCreate,omitempty"`
	DiskDelete             *DeployerSetResultItemGeneral    `json:"diskDelete,omitempty"`
	DeploymentDeploy       *DeployerSetResultItemDeploy     `json:"deploymentDeploy,omitempty"`
	DeploymentDelete       *DeployerSetResultItemGeneral    `json:"deploymentDelete,omitempty"`
	DeploymentPause        *DeployerSetResultItemDeployment `json:"deploymentPause,omitempty"`
	DeploymentCleanup      *DeployerSetResultItemDeployment `json:"deploymentCleanup,omitempty"`
	RouteCreate            *DeployerSetResultItemGeneral    `json:"routeCreate,omitempty"`
	RouteDelete            *DeployerSetResultItemGeneral    `json:"routeDelete,omitempty"`
}

type DeployerSetResultItemGeneral struct {
	ID int64 `json:"id"`
}

type DeployerSetResultItemDeploy struct {
	ID       int64 `json:"id"`
	Revision int64 `json:"revision"`
	Success  bool  `json:"success"`
	NodePort *int  `json:"nodePort,omitempty"`
}

type DeployerSetResultItemDeployment struct {
	ID       int64 `json:"id"`
	Revision int64 `json:"revision"`
}
