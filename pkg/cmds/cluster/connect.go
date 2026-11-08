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
	"go.bytebuilders.dev/cli/pkg/akp"
	"go.bytebuilders.dev/cli/pkg/config"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
)

func newCmdConnect(f *config.Factory) *cobra.Command {
	req := &clusterv2.ConnectClusterRequest{Cluster: &clusterv2.ClusterRef{}}
	var kubeConfigPath string
	cmd := &cobra.Command{
		Use:               "connect",
		Short:             "Connect with a cluster imported by peers",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if kubeConfigPath != "" {
				data, err := os.ReadFile(kubeConfigPath)
				if err != nil {
					return fmt.Errorf("failed to read Kubeconfig file. Reason: %w", err)
				}
				req.Kubeconfig = string(data)
			}
			err := connectCluster(cmd, f, req)
			if err != nil {
				if akp.IsNotFound(err) {
					fmt.Println("Provided cluster does not exist. Please provide a valid cluster name.")
					return nil
				}
				return fmt.Errorf("failed to connect with cluster. Reason: %w", err)
			}
			fmt.Println("Successfully connected to the cluster")
			return nil
		},
	}
	cmd.Flags().StringVar(&req.Cluster.Name, "name", "", "Name of the cluster to get")
	cmd.Flags().StringVar(&req.Credential, "credential", "", "Name of the credential to use to connect with the cluster")
	cmd.Flags().StringVar(&kubeConfigPath, "kubeconfig", "", "Path of the kubeconfig file")
	return cmd
}

func connectCluster(cmd *cobra.Command, f *config.Factory, req *clusterv2.ConnectClusterRequest) error {
	c, err := f.Clients()
	if err != nil {
		return err
	}
	req.Cluster.Owner, err = c.Owner(cmd.Context())
	if err != nil {
		return err
	}
	_, err = c.Lifecycle.ConnectCluster(cmd.Context(), connect.NewRequest(req))
	return err
}
