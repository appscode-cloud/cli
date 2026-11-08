/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package akp builds the clients of the AKP v2 API, served under /api/v2.
package akp

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	clusterv2 "go.bytebuilders.dev/b3/apis/gen/go/akp/cluster/v2"
	"go.bytebuilders.dev/b3/apis/gen/go/akp/cluster/v2/clusterv2connect"
	userv2 "go.bytebuilders.dev/b3/apis/gen/go/akp/user/v2"
	"go.bytebuilders.dev/b3/apis/gen/go/akp/user/v2/userv2connect"

	"connectrpc.com/connect"
	"github.com/nats-io/nats.go"
)

const csrfCookie = "_csrf"

// Credentials are what the CLI signs in to AKP with.
type Credentials struct {
	Token    string
	Username string
	Password string
	Cookies  []http.Cookie
}

type Clients struct {
	// Organization is the account to act on; empty means the caller's own.
	Organization string

	Users     userv2connect.UserServiceClient
	Nats      userv2connect.NatsServiceClient
	Clusters  clusterv2connect.ClusterServiceClient
	Lifecycle clusterv2connect.LifecycleServiceClient
}

func NewClients(endpoint, organization string, cred Credentials) *Clients {
	baseURL := strings.TrimSuffix(endpoint, "/") + "/api/v2"
	hc := http.DefaultClient
	opts := []connect.ClientOption{connect.WithInterceptors(credentialsInterceptor(cred))}
	return &Clients{
		Organization: organization,
		Users:        userv2connect.NewUserServiceClient(hc, baseURL, opts...),
		Nats:         userv2connect.NewNatsServiceClient(hc, baseURL, opts...),
		Clusters:     clusterv2connect.NewClusterServiceClient(hc, baseURL, opts...),
		Lifecycle:    clusterv2connect.NewLifecycleServiceClient(hc, baseURL, opts...),
	}
}

func credentialsInterceptor(cred Credentials) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if req.Spec().IsClient {
				setCredentials(req.Header(), cred)
			}
			return next(ctx, req)
		}
	}
}

func setCredentials(h http.Header, cred Credentials) {
	if cred.Token != "" {
		h.Set("Authorization", "token "+cred.Token)
	}
	// Basic auth replaces the token, as it did with the AKP v1 API client.
	if cred.Username != "" && cred.Password != "" {
		h.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(cred.Username+":"+cred.Password)))
	}
	if len(cred.Cookies) == 0 {
		return
	}
	cookies := make([]string, 0, len(cred.Cookies))
	for _, c := range cred.Cookies {
		cookies = append(cookies, (&http.Cookie{Name: c.Name, Value: c.Value}).String())
		if c.Name == csrfCookie {
			// AKP checks the CSRF token of a session caller.
			h.Set("X-Csrf-Token", c.Value)
		}
	}
	h.Set("Cookie", strings.Join(cookies, "; "))
}

// Owner is the account the CLI acts on: the chosen organization, else the caller.
func (c *Clients) Owner(ctx context.Context) (string, error) {
	if c.Organization != "" {
		return c.Organization, nil
	}
	resp, err := c.Users.GetAuthenticatedUser(ctx, connect.NewRequest(&userv2.GetAuthenticatedUserRequest{}))
	if err != nil {
		return "", fmt.Errorf("organization name wasn't provided. Automatic user detection failed with error: %w", err)
	}
	return resp.Msg.GetUser().GetUsername(), nil
}

// NatsConnection connects to AKP's NATS with the caller's credentials.
func (c *Clients) NatsConnection(ctx context.Context, name string) (*nats.Conn, error) {
	resp, err := c.Nats.GetNatsCredentials(ctx, connect.NewRequest(&userv2.GetNatsCredentialsRequest{}))
	if err != nil {
		return nil, err
	}
	if len(resp.Msg.GetEndpoints()) == 0 {
		return nil, errors.New("AKP answered no NATS endpoint")
	}

	credFile, err := os.CreateTemp("", "nats-*.creds")
	if err != nil {
		return nil, err
	}
	defer os.Remove(credFile.Name()) // nolint:errcheck
	_, err = credFile.Write(resp.Msg.GetCredentials())
	if cerr := credFile.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return nil, err
	}

	return nats.Connect(
		resp.Msg.GetEndpoints()[0],
		nats.Name(name),
		nats.UserCredentials(credFile.Name()),
		nats.NoReconnect(),
	)
}

// ClusterStatus is a cluster with its live status.
func (c *Clients) ClusterStatus(ctx context.Context, owner, name string) (*clusterv2.Cluster, error) {
	resp, err := c.Lifecycle.GetClusterStatus(ctx, connect.NewRequest(&clusterv2.GetClusterStatusRequest{
		Cluster: &clusterv2.ClusterRef{Owner: owner, Name: name},
	}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetCluster(), nil
}

func IsNotFound(err error) bool {
	return connect.CodeOf(err) == connect.CodeNotFound
}
