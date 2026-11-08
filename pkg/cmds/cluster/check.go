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
	"os"

	clusterv2 "go.bytebuilders.dev/b3/apis/gen/go/akp/cluster/v2"
	"go.bytebuilders.dev/cli/pkg/config"
	"go.bytebuilders.dev/cli/pkg/printer"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	rsapi "kmodules.xyz/resource-metadata/apis/meta/v1alpha1"
)

func newCmdCheck(f *config.Factory) *cobra.Command {
	provider := &clusterv2.ProviderOptions{}
	var kubeConfigPath string
	cmd := &cobra.Command{
		Use:               "check",
		Short:             "Check whether a cluster has been imported already or not",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if kubeConfigPath != "" {
				data, err := os.ReadFile(kubeConfigPath)
				if err != nil {
					return fmt.Errorf("failed to read Kubeconfig file. Reason: %w", err)
				}
				provider.Kubeconfig = string(data)
			}
			cluster, err := checkClusterExistence(cmd, f, provider)
			if err != nil {
				return fmt.Errorf("failed to check cluster existence. Reason: %w", err)
			}
			if cluster.GetStatus().GetPhase() == string(rsapi.ClusterPhaseNotImported) {
				fmt.Println("Cluster hasn't been imported yet.")
				return nil
			}
			return printer.PrintCluster(cluster)
		},
	}
	addProviderFlags(cmd, provider, &kubeConfigPath)
	return cmd
}

func addProviderFlags(cmd *cobra.Command, provider *clusterv2.ProviderOptions, kubeConfigPath *string) {
	cmd.Flags().StringVar(&provider.Name, "provider", "", "Name of the cluster provider")
	cmd.Flags().StringVar(&provider.Credential, "credential", "", "Name of the credential with access to the provider APIs")
	cmd.Flags().StringVar(&provider.ClusterId, "id", "", "Provider specific cluster ID")
	cmd.Flags().StringVar(&provider.Project, "project", "", "Project where the cluster belong (use for GKE)")
	cmd.Flags().StringVar(&provider.Region, "region", "", "Region or location of the cluster")
	cmd.Flags().StringVar(&provider.ResourceGroup, "resource-group", "", "Resource group of the cluster (use for AKS)")
	cmd.Flags().StringVar(kubeConfigPath, "kubeconfig", "", "Path of the kubeconfig file")
}

func checkClusterExistence(cmd *cobra.Command, f *config.Factory, provider *clusterv2.ProviderOptions) (*clusterv2.Cluster, error) {
	c, err := f.Clients()
	if err != nil {
		return nil, err
	}
	owner, err := c.Owner(cmd.Context())
	if err != nil {
		return nil, err
	}
	resp, err := c.Lifecycle.CheckCluster(cmd.Context(), connect.NewRequest(&clusterv2.CheckClusterRequest{
		Owner:    owner,
		Provider: provider,
	}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetCluster(), nil
}
