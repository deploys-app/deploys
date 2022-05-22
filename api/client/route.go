package client

import (
	"context"

	"github.com/deploys-app/deploys/api"
)

type routeClient struct {
	inv invoker
}

func (c routeClient) Create(ctx context.Context, m *api.RouteCreate) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "route.create", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c routeClient) CreateV2(ctx context.Context, m *api.RouteCreateV2) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "route.createV2", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c routeClient) Get(ctx context.Context, m *api.RouteGet) (*api.RouteItem, error) {
	var res api.RouteItem
	err := c.inv.invoke(ctx, "route.get", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c routeClient) List(ctx context.Context, m *api.RouteList) (*api.RouteListResult, error) {
	var res api.RouteListResult
	err := c.inv.invoke(ctx, "route.list", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c routeClient) ListV2(ctx context.Context, m *api.RouteList) (*api.RouteListV2Result, error) {
	var res api.RouteListV2Result
	err := c.inv.invoke(ctx, "route.listV2", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c routeClient) Delete(ctx context.Context, m *api.RouteDelete) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "route.delete", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
