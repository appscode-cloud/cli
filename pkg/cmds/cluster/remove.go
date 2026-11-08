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
	"context"
	"fmt"

	clusterv2 "go.bytebuilders.dev/b3/apis/gen/go/akp/cluster/v2"
	"go.bytebuilders.dev/cli/pkg/akp"
	"go.bytebuilders.dev/cli/pkg/config"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
)

func newCmdRemove(f *config.Factory) *cobra.Command {
	req := &clusterv2.RemoveClusterRequest{
		Cluster:    &clusterv2.ClusterRef{},
		Components: &clusterv2.ComponentOptions{},
	}
	cmd := &cobra.Command{
		Use:               "remove",
		Short:             "Remove a cluster from ACE platform",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !req.Components.AllFeatures {
				req.Components.FeatureSets = defaultFeatureSets()
			}
			err := removeCluster(cmd, f, req)
			if err != nil {
				if akp.IsNotFound(err) {
					fmt.Println("Cluster has been removed already.")
					return nil
				}
				return fmt.Errorf("failed to remove cluster. Reason: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&req.Cluster.Name, "name", "", "Name of the cluster to get")
	cmd.Flags().BoolVar(&req.Components.FluxCd, "remove-fluxcd", true, "Specify whether to remove FluxCD or not (default true).")
	cmd.Flags().BoolVar(&req.Components.AllFeatures, "all-features", false, "Remove all features")
	return cmd
}

func removeCluster(cmd *cobra.Command, f *config.Factory, req *clusterv2.RemoveClusterRequest) error {
	fmt.Println("Removing cluster......")
	return runOperation(cmd, f, "removal", func(ctx context.Context, c *akp.Clients, owner string, op *clusterv2.OperationOptions) error {
		req.Cluster.Owner = owner
		req.Operation = op
		_, err := c.Lifecycle.RemoveCluster(ctx, connect.NewRequest(req))
		return err
	})
}
