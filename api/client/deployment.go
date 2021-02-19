package client

import (
	"context"

	"moonrhythm/deploys/api"
)

type deploymentClient struct {
	inv invoker
}

func (c deploymentClient) Deploy(ctx context.Context, m api.DeploymentDeploy) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "deployment.deploy", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c deploymentClient) List(ctx context.Context, m api.DeploymentList) (*api.DeploymentListResult, error) {
	var res api.DeploymentListResult
	err := c.inv.invoke(ctx, "deployment.list", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c deploymentClient) Get(ctx context.Context, m api.DeploymentGet) (*api.DeploymentGetResult, error) {
	var res api.DeploymentGetResult
	err := c.inv.invoke(ctx, "deployment.get", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c deploymentClient) Resume(ctx context.Context, m api.DeploymentResume) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "deployment.resume", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c deploymentClient) Pause(ctx context.Context, m api.DeploymentPause) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "deployment.pause", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c deploymentClient) Rollback(ctx context.Context, m api.DeploymentRollback) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "deployment.rollback", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c deploymentClient) Delete(ctx context.Context, m api.DeploymentDelete) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "deployment.delete", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c deploymentClient) MapDomain(ctx context.Context, m api.DeploymentMapDomain) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "deployment.mapDomain", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
