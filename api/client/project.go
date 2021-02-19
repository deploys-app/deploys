package client

import (
	"context"

	"moonrhythm/deploys/api"
)

type projectClient struct {
	inv invoker
}

func (c projectClient) Create(ctx context.Context, m api.ProjectCreate) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "project.create", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c projectClient) Update(ctx context.Context, m api.ProjectUpdate) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "project.update", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c projectClient) Get(ctx context.Context, m api.ProjectGet) (*api.ProjectGetResult, error) {
	var res api.ProjectGetResult
	err := c.inv.invoke(ctx, "project.get", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c projectClient) List(ctx context.Context, m api.Empty) (*api.ProjectListResult, error) {
	var res api.ProjectListResult
	err := c.inv.invoke(ctx, "project.list", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c projectClient) Usage(ctx context.Context, m api.ProjectUsage) (*api.ProjectUsageResult, error) {
	var res api.ProjectUsageResult
	err := c.inv.invoke(ctx, "project.usage", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
