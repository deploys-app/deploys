package client

import (
	"context"

	"github.com/deploys-app/deploys/api"
)

type locationClient struct {
	inv invoker
}

func (c locationClient) List(ctx context.Context, m *api.Empty) (*api.LocationListResult, error) {
	var res api.LocationListResult
	err := c.inv.invoke(ctx, "location.list", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c locationClient) Get(ctx context.Context, m *api.LocationGet) (*api.LocationItem, error) {
	var res api.LocationItem
	err := c.inv.invoke(ctx, "location.get", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
