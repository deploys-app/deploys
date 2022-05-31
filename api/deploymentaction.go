package api

//go:generate stringer -type=DeploymentAction -linecomment
type DeploymentAction int

const (
	_                      DeploymentAction = iota
	DeploymentActionDeploy                  // deploy
	DeploymentActionDelete                  // delete
	DeploymentActionPause                   // pause
)
