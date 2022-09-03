package client

import (
	"context"

	"github.com/deploys-app/deploys/api"
)

type domainClient struct {
	inv invoker
}

func (c domainClient) Create(ctx context.Context, m *api.DomainCreate) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "domain.create", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c domainClient) Get(ctx context.Context, m *api.DomainGet) (*api.DomainItem, error) {
	var res api.DomainItem
	err := c.inv.invoke(ctx, "domain.get", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c domainClient) List(ctx context.Context, m *api.DomainList) (*api.DomainListResult, error) {
	var res api.DomainListResult
	err := c.inv.invoke(ctx, "domain.list", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c domainClient) Delete(ctx context.Context, m *api.DomainDelete) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "domain.delete", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c domainClient) PurgeCache(ctx context.Context, m *api.DomainPurgeCache) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "domain.purgeCache", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
