package client

import (
	"context"

	"github.com/deploys-app/deploys/api"
)

type emailClient struct {
	inv invoker
}

func (c emailClient) Send(ctx context.Context, m *api.EmailSend) (*api.Empty, error) {
	var res api.Empty
	err := c.inv.invoke(ctx, "email.send", m, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
