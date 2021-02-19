package client

import (
	"context"

	"moonrhythm/deploys/api"
)

type collectorClient struct {
	inv invoker
}

func (c collectorClient) Location(ctx context.Context, m api.CollectorLocation) (*api.CollectorLocationResult, error) {
	var res api.CollectorLocationResult
	err := c.inv.invoke(ctx, "collector.location", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c collectorClient) SetProjectUsage(ctx context.Context, m api.CollectorSetProjectUsage) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "collector.setProjectUsage", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c collectorClient) SetDeploymentUsage(ctx context.Context, m api.CollectorSetDeploymentUsage) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "collector.setDeploymentUsage", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
