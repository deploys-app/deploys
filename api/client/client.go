package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/deploys-app/deploys/api"
)

const endpoint = "https://api.deploys.app/"

type Error struct {
	Message string `json:"message"`
}

func (err *Error) Error() string {
	return err.Message
}

func (err *Error) apiError() error {
	// try to convert error message to api error
	// TODO: find a best way to handle this, error code ?
	for _, e := range api.AllErrors {
		if e.Error() == err.Message {
			return e
		}
	}
	return err
}

type invoker interface {
	invoke(ctx context.Context, api string, r interface{}, res interface{}) error
}

type Client struct {
	HTTPClient *http.Client
	Endpoint   string
	Auth       func(r *http.Request)
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient == nil {
		return http.DefaultClient
	}
	return c.HTTPClient
}

func (c *Client) endpoint() string {
	if c.Endpoint == "" {
		return endpoint
	}
	return strings.TrimSuffix(c.Endpoint, "/") + "/"
}

func (c *Client) Me() api.Me {
	return meClient{c}
}

func (c *Client) Location() api.Location {
	return locationClient{c}
}

func (c *Client) Project() api.Project {
	return projectClient{c}
}

func (c *Client) Billing() api.Billing {
	return billingClient{c}
}

func (c *Client) ServiceAccount() api.ServiceAccount {
	return serviceAccountClient{c}
}

func (c *Client) Role() api.Role {
	return roleClient{c}
}

func (c *Client) Deployment() api.Deployment {
	return deploymentClient{c}
}

func (c *Client) Route() api.Route {
	return routeClient{c}
}

func (c *Client) Disk() api.Disk {
	return diskClient{c}
}

func (c *Client) PullSecret() api.PullSecret {
	return pullSecretClient{c}
}

func (c *Client) WorkloadIdentity() api.WorkloadIdentity {
	return workloadIdentityClient{c}
}

func (c *Client) Collector() api.Collector {
	return collectorClient{c}
}

func (c *Client) invoke(ctx context.Context, api string, r interface{}, res interface{}) error {
	if err := validRequest(r); err != nil {
		return err
	}

	var reqBody bytes.Buffer
	err := json.NewEncoder(&reqBody).Encode(r)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.endpoint()+api, &reqBody)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")
	if c.Auth != nil {
		c.Auth(req)
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("not ok")
	}
	defer io.Copy(io.Discard, resp.Body)

	var errMsg Error
	var respBody struct {
		OK     bool        `json:"ok"`
		Result interface{} `json:"result"`
		Error  interface{} `json:"error"`
	}
	respBody.Result = res
	respBody.Error = &errMsg

	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		return err
	}

	if !respBody.OK {
		return errMsg.apiError()
	}
	return nil
}

func validRequest(r interface{}) error {
	if r == nil {
		return nil
	}

	if r, ok := r.(interface {
		Valid() error
	}); ok {
		return r.Valid()
	}

	return nil
}
