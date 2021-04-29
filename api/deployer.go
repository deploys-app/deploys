package api

import (
	"context"
)

type Deployer interface {
	GetCommands(ctx context.Context, m *DeployerGetCommands) (*GetCommandsResult, error)
	SetResult(ctx context.Context, m *DeployerSetResult) (*Empty, error)
}

type DeployerGetCommands struct {
	Location string
}

type GetCommandsResult struct {
	Deployments        []*DeployerDeploymentCommand
	PullSecrets        []*DeployerPullSecretCommand
	Disks              []*DeployerDiskCommand
	WorkloadIdentities []*DeployerWorkloadIdentityCommand
	Routes             []*DeployerRouteCommand
}

type DeployerSetResult struct {
	Command  string
	Location string
	ID       int64
	Revision int64 // for Deployment
	Status   Status
}

type DeployerDeploymentCommand struct {
	ID        int64
	ProjectID int64
	Name      string
	Spec      DeploymentSpec
}

type DeploymentSpec struct {
	Image                string
	Env                  map[string]string
	Command              []string
	Args                 []string
	WorkloadIdentityName string
	MinReplicas          int
	MaxReplicas          int
	Port                 int
	Protocol             DeploymentProtocol
	Schedule             string
	Annotations          map[string]string
	CPU                  string
	Memory               string
	PullSecretName       string
	DiskName             string
	DiskMountPath        string
	DiskSubPath          string
	MountData            map[string]string // file path => data
}

type DeployerPullSecretCommand struct {
	ID        int64
	ProjectID int64
	Name      string
	Value     string
}

type DeployerDiskCommand struct {
	ID        int64
	ProjectID int64
	Name      string
	Size      int64
}

type DeployerWorkloadIdentityCommand struct {
	ID        int64
	ProjectID int64
	Name      string
	GSA       string
}

type DeployerRouteCommand struct {
	ID             int64
	ProjectID      int64
	Domain         string
	Path           string
	DeploymentName string
}
