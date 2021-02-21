package client

import (
	"context"

	"github.com/deploys-app/deploys/api"
)

type serviceAccountClient struct {
	inv invoker
}

func (c serviceAccountClient) Create(ctx context.Context, m *api.ServiceAccountCreate) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "serviceaccount.create", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c serviceAccountClient) Get(ctx context.Context, m *api.ServiceAccountGet) (*api.ServiceAccountGetResult, error) {
	var res api.ServiceAccountGetResult
	err := c.inv.invoke(ctx, "serviceaccount.get", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c serviceAccountClient) List(ctx context.Context, m *api.ServiceAccountList) (*api.ServiceAccountListResult, error) {
	var res api.ServiceAccountListResult
	err := c.inv.invoke(ctx, "serviceaccount.list", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c serviceAccountClient) Update(ctx context.Context, m *api.ServiceAccountUpdate) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "serviceaccount.update", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c serviceAccountClient) Delete(ctx context.Context, m *api.ServiceAccountDelete) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "serviceaccount.delete", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c serviceAccountClient) CreateKey(ctx context.Context, m *api.ServiceAccountCreateKey) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "serviceaccount.createKey", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c serviceAccountClient) DeleteKey(ctx context.Context, m *api.ServiceAccountDeleteKey) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "serviceaccount.deleteKey", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
