package client

import (
	"context"

	"github.com/deploys-app/deploys/api"
)

type deployerClient struct {
	inv invoker
}

func (c deployerClient) GetCommands(ctx context.Context, m *api.DeployerGetCommands) (*api.GetCommandsResult, error) {
	var res api.GetCommandsResult
	err := c.inv.invoke(ctx, "deployer.getCommands", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c deployerClient) SetResult(ctx context.Context, m *api.DeployerSetResult) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "deployer.setResult", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
