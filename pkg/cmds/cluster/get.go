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

	clusterv2 "go.bytebuilders.dev/b3/apis/gen/go/akp/cluster/v2"
	"go.bytebuilders.dev/cli/pkg/akp"
	"go.bytebuilders.dev/cli/pkg/config"
	"go.bytebuilders.dev/cli/pkg/printer"

	"github.com/spf13/cobra"
)

func newCmdGet(f *config.Factory) *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:               "get",
		Short:             "Get a particular cluster information",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cluster, err := getCluster(cmd, f, name)
			if err != nil {
				if akp.IsNotFound(err) {
					fmt.Println("Cluster does not exist.")
					return nil
				}
				return fmt.Errorf("failed to get the cluster information. Reason: %w", err)
			}
			return printer.PrintCluster(cluster)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Name of the cluster to get")
	return cmd
}

func getCluster(cmd *cobra.Command, f *config.Factory, name string) (*clusterv2.Cluster, error) {
	c, err := f.Clients()
	if err != nil {
		return nil, err
	}
	owner, err := c.Owner(cmd.Context())
	if err != nil {
		return nil, err
	}
	return c.ClusterStatus(cmd.Context(), owner, name)
}
