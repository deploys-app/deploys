package client

import (
	"context"

	"github.com/deploys-app/deploys/api"
)

type deployerClient struct {
	inv invoker
}

func (c deployerClient) GetLocation(ctx context.Context, m *api.Empty) (*api.LocationItem, error) {
	var res api.LocationItem
	err := c.inv.invoke(ctx, "deployer.getLocation", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c deployerClient) IsDomainActive(ctx context.Context, m *api.DeployerIsDomainActive) (bool, error) {
	var res bool
	err := c.inv.invoke(ctx, "deployer.isDomainActive", m, &res)
	if err != nil {
		return false, err
	}
	return res, nil
}

func (c deployerClient) GetCommands(ctx context.Context, m *api.Empty) (*api.GetCommandsResult, error) {
	var res api.GetCommandsResult
	err := c.inv.invoke(ctx, "deployer.getCommands", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c deployerClient) SetResults(ctx context.Context, m *api.DeployerSetResult) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "deployer.setResults", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
