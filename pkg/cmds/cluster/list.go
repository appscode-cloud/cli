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

package cluster

import (
	"fmt"
	"strings"

	clusterv2 "go.bytebuilders.dev/b3/apis/gen/go/akp/cluster/v2"
	"go.bytebuilders.dev/cli/pkg/config"
	"go.bytebuilders.dev/cli/pkg/printer"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
)

func newCmdList(f *config.Factory) *cobra.Command {
	var provider string
	cmd := &cobra.Command{
		Use:               "list",
		Short:             "List cluster managed by ACE platform",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			clusters, err := listClusters(cmd, f, provider)
			if err != nil {
				return fmt.Errorf("failed to list clusters. Reason: %w", err)
			}
			if len(clusters) == 0 {
				fmt.Println("No cluster found.")
				return nil
			}
			return printer.PrintClusterList(clusters)
		},
	}
	cmd.Flags().StringVar(&provider, "provider", "", "List cluster only for this provider")
	return cmd
}

func listClusters(cmd *cobra.Command, f *config.Factory, provider string) ([]*clusterv2.Cluster, error) {
	ctx := cmd.Context()
	c, err := f.Clients()
	if err != nil {
		return nil, err
	}
	owner, err := c.Owner(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := c.Clusters.ListClusters(ctx, connect.NewRequest(&clusterv2.ListClustersRequest{Owner: owner}))
	if err != nil {
		return nil, err
	}

	var clusters []*clusterv2.Cluster
	for _, cluster := range resp.Msg.GetClusters() {
		// The API lists every provider's clusters.
		if provider != "" && !strings.EqualFold(cluster.GetProvider(), provider) {
			continue
		}
		// The list carries stored records; the live status is asked per cluster.
		live, err := c.ClusterStatus(ctx, owner, cluster.GetName())
		if err != nil {
			return nil, err
		}
		cluster.Status = live.GetStatus()
		clusters = append(clusters, cluster)
	}
	return clusters, nil
}
