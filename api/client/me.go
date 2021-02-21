package client

import (
	"context"

	"github.com/deploys-app/deploys/api"
)

type meClient struct {
	inv invoker
}

func (c meClient) Get(ctx context.Context, m *api.Empty) (*api.MeItem, error) {
	var res api.MeItem
	err := c.inv.invoke(ctx, "me.get", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c meClient) Authorized(ctx context.Context, m *api.MeAuthorized) (*api.MeAuthorizedResult, error) {
	var res api.MeAuthorizedResult
	err := c.inv.invoke(ctx, "me.authorized", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c meClient) UploadKYCDocument(ctx context.Context, _ *api.MeUploadKYCDocument) (*api.MeUploadKYCDocumentResult, error) {
	return nil, nil
}
