package api

import "encoding/json"

//go:generate stringer -type=DeploymentAction -linecomment
type DeploymentAction int

const (
	_                      DeploymentAction = iota
	DeploymentActionDeploy                  // deploy
	DeploymentActionDelete                  // delete
	DeploymentActionPause                   // pause
)

func (a DeploymentAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

func (a *DeploymentAction) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	*a = DeploymentAction(0)

	for _, x := range []DeploymentAction{DeploymentActionDeploy, DeploymentActionDelete, DeploymentActionPause} {
		if x.String() == s {
			*a = x
			return nil
		}
	}
	return nil
}
