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
	"os"
	"strings"

	clusterv2 "go.bytebuilders.dev/b3/apis/gen/go/akp/cluster/v2"
	"go.bytebuilders.dev/cli/pkg/akp"
	"go.bytebuilders.dev/cli/pkg/config"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
)

func newCmdImport(f *config.Factory) *cobra.Command {
	req := &clusterv2.ImportClusterRequest{
		Names:      &clusterv2.ClusterNames{},
		Provider:   &clusterv2.ProviderOptions{},
		Components: &clusterv2.ComponentOptions{},
	}
	var featureSet map[string]string
	var kubeConfigPath string
	cmd := &cobra.Command{
		Use:               "import",
		Short:             "Import a cluster to ACE platform",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if kubeConfigPath != "" {
				data, err := os.ReadFile(kubeConfigPath)
				if err != nil {
					return fmt.Errorf("failed to read Kubeconfig file. Reason: %w", err)
				}
				req.Provider.Kubeconfig = string(data)
			}

			req.Components.FeatureSets = getFeatureSetsInfo(featureSet)

			err := importCluster(cmd, f, req)
			if err != nil {
				return fmt.Errorf("failed to import cluster. Reason: %w", err)
			}
			return nil
		},
	}
	addProviderFlags(cmd, req.Provider, &kubeConfigPath)

	cmd.Flags().StringVar(&req.Names.DisplayName, "display-name", "", "Display name of the cluster")
	cmd.Flags().StringVar(&req.Names.Name, "name", "", "Unique name across all imported clusters of all provider")
	cmd.Flags().BoolVar(&req.Components.FluxCd, "install-fluxcd", true, "Specify whether to install FluxCD or not (default true).")
	cmd.Flags().BoolVar(&req.Components.AllFeatures, "all-features", false, "Install all features")
	cmd.Flags().StringToStringVar(&featureSet, "featureset", featureSet, "List of features")
	return cmd
}

func importCluster(cmd *cobra.Command, f *config.Factory, req *clusterv2.ImportClusterRequest) error {
	fmt.Println("Importing cluster......")
	return runOperation(cmd, f, "the import", func(ctx context.Context, c *akp.Clients, owner string, op *clusterv2.OperationOptions) error {
		req.Owner = owner
		req.Operation = op
		_, err := c.Lifecycle.ImportCluster(ctx, connect.NewRequest(req))
		return err
	})
}

func getFeatureSetsInfo(featureSets map[string]string) []*clusterv2.FeatureSet {
	if len(featureSets) == 0 {
		return defaultFeatureSets()
	}

	desiredFeatureSets := make([]*clusterv2.FeatureSet, 0, len(featureSets))
	for key, val := range featureSets {
		desiredFeatureSets = append(desiredFeatureSets, &clusterv2.FeatureSet{
			Name:     key,
			Features: strings.Split(val, ","),
		})
	}
	return desiredFeatureSets
}
