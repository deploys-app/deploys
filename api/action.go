package api

//go:generate stringer -type=Action -linecomment
type Action int

const (
	_      Action = iota
	Create        // create
	Delete        // delete
)

type DeploymentAction int

const (
	_ DeploymentAction = iota
	DeploymentActionDeploy
	DeploymentActionDelete
	DeploymentActionPause
)

func (a DeploymentAction) String() string {
	switch a {
	case DeploymentActionDeploy:
		return "deploy"
	case DeploymentActionDelete:
		return "delete"
	case DeploymentActionPause:
		return "pause"
	default:
		return ""
	}
}
