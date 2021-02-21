package client

import (
	"context"

	"github.com/deploys-app/deploys/api"
)

type billingClient struct {
	inv invoker
}

func (c billingClient) Create(ctx context.Context, m *api.BillingCreate) (*api.BillingCreateResult, error) {
	var res api.BillingCreateResult
	err := c.inv.invoke(ctx, "billing.create", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c billingClient) List(ctx context.Context, m *api.Empty) (*api.BillingListResult, error) {
	var res api.BillingListResult
	err := c.inv.invoke(ctx, "billing.list", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c billingClient) Delete(ctx context.Context, m *api.BillingDelete) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "billing.delete", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c billingClient) Get(ctx context.Context, m *api.BillingGet) (*api.BillingItem, error) {
	var res api.BillingItem
	err := c.inv.invoke(ctx, "billing.get", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c billingClient) Update(ctx context.Context, m *api.BillingUpdate) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "billing.update", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c billingClient) Report(ctx context.Context, m *api.BillingReport) (*api.BillingReportResult, error) {
	var res api.BillingReportResult
	err := c.inv.invoke(ctx, "billing.report", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c billingClient) SKUs(ctx context.Context, m *api.Empty) (*api.BillingSKUs, error) {
	var res api.BillingSKUs
	err := c.inv.invoke(ctx, "billing.skus", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c billingClient) Project(ctx context.Context, m *api.BillingProject) (*api.BillingProjectResult, error) {
	var res api.BillingProjectResult
	err := c.inv.invoke(ctx, "billing.project", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
