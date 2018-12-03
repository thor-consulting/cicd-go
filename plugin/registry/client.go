package registry

import (
	"context"

	"github.com/banzaicloud/cicd-go/cicd"
	"github.com/banzaicloud/cicd-go/plugin/internal/client"
)

// Client returns a new plugin client.
func Client(endpoint, secret string, skipverify bool) Plugin {
	client := client.New(endpoint, secret, skipverify)
	client.Accept = V1
	return &pluginClient{
		client: client,
	}
}

type pluginClient struct {
	client *client.Client
}

func (c *pluginClient) List(ctx context.Context, in *Request) ([]*cicd.Registry, error) {
	res := []*cicd.Registry{}
	err := c.client.Do(in, &res)
	return res, err
}