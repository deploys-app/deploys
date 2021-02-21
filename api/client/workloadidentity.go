package client

import (
	"context"

	"github.com/deploys-app/deploys/api"
)

type workloadIdentityClient struct {
	inv invoker
}

func (c workloadIdentityClient) Create(ctx context.Context, m *api.WorkloadIdentityCreate) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "workloadidentity.create", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c workloadIdentityClient) Get(ctx context.Context, m *api.WorkloadIdentityGet) (*api.WorkloadIdentityItem, error) {
	var res api.WorkloadIdentityItem
	err := c.inv.invoke(ctx, "workloadidentity.get", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c workloadIdentityClient) List(ctx context.Context, m *api.WorkloadIdentityList) (*api.WorkloadIdentityListResult, error) {
	var res api.WorkloadIdentityListResult
	err := c.inv.invoke(ctx, "workloadidentity.list", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c workloadIdentityClient) Delete(ctx context.Context, m *api.WorkloadIdentityDelete) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "workloadidentity.delete", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
