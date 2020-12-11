// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package galiaslookup

import (
	"context"

	"github.com/liraxapp/avalanchego/ids"
	"github.com/liraxapp/avalanchego/snow"
	"github.com/liraxapp/avalanchego/vms/rpcchainvm/galiaslookup/galiaslookupproto"
)

var (
	_ snow.AliasLookup = &Client{}
)

// Client is an implementation of a messenger channel that talks over RPC.
type Client struct {
	client galiaslookupproto.AliasLookupClient
}

// NewClient returns a alias lookup instance connected to a remote alias lookup instance
func NewClient(client galiaslookupproto.AliasLookupClient) *Client {
	return &Client{client: client}
}

// Lookup ...
func (c *Client) Lookup(alias string) (ids.ID, error) {
	resp, err := c.client.Lookup(context.Background(), &galiaslookupproto.LookupRequest{
		Alias: alias,
	})
	if err != nil {
		return ids.ID{}, err
	}
	return ids.ToID(resp.Id)
}

// PrimaryAlias ...
func (c *Client) PrimaryAlias(id ids.ID) (string, error) {
	resp, err := c.client.PrimaryAlias(context.Background(), &galiaslookupproto.PrimaryAliasRequest{
		Id: id[:],
	})
	if err != nil {
		return "", err
	}
	return resp.Alias, nil
}
