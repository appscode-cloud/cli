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
	"strings"
	"sync"

	"go.bytebuilders.dev/cli/pkg/config"
	"go.bytebuilders.dev/cli/pkg/printer"
	clustermodel "go.bytebuilders.dev/resource-model/apis/cluster"

	"github.com/rs/xid"
	"github.com/spf13/cobra"
)

func newCmdImport(f *config.Factory) *cobra.Command {
	opts := clustermodel.ImportOptions{}
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
				opts.Provider.KubeConfig = string(data)
			}

			opts.Components.FeatureSets = getFeatureSetsInfo(featureSet)

			err := importCluster(f, opts)
			if err != nil {
				return fmt.Errorf("failed to import cluster. Reason: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.Provider.Name, "provider", "", "Name of the cluster provider")
	cmd.Flags().StringVar(&opts.Provider.Credential, "credential", "", "Name of the credential with access to the provider APIs")
	cmd.Flags().StringVar(&opts.Provider.ClusterID, "id", "", "Provider specific cluster ID")
	cmd.Flags().StringVar(&opts.Provider.Project, "project", "", "Project where the cluster belong (use for GKE)")
	cmd.Flags().StringVar(&opts.Provider.Region, "region", "", "Region or location of the cluster")
	cmd.Flags().StringVar(&opts.Provider.ResourceGroup, "resource-group", "", "Resource group of the cluster (use for AKS)")
	cmd.Flags().StringVar(&kubeConfigPath, "kubeconfig", "", "Path of the kubeconfig file")

	cmd.Flags().StringVar(&opts.BasicInfo.DisplayName, "display-name", "", "Display name of the cluster")
	cmd.Flags().StringVar(&opts.BasicInfo.Name, "name", "", "Unique name across all imported clusters of all provider")
	cmd.Flags().BoolVar(&opts.Components.FluxCD, "install-fluxcd", true, "Specify whether to install FluxCD or not (default true).")
	cmd.Flags().BoolVar(&opts.Components.AllFeatures, "all-features", false, "Install all features")
	cmd.Flags().StringToStringVar(&featureSet, "featureset", featureSet, "List of features")
	return cmd
}

func importCluster(f *config.Factory, opts clustermodel.ImportOptions) error {
	fmt.Println("Importing cluster......")
	c, err := f.Client()
	if err != nil {
		return err
	}
	nc, err := c.NewNatsConnection("ace-cli")
	if err != nil {
		return err
	}
	defer nc.Close()

	responseID := xid.New().String()
	wg := sync.WaitGroup{}
	wg.Add(1)
	done := f.Canceller()
	go func() {
		err := printer.PrintNATSJobSteps(&wg, nc, responseID, done)
		if err != nil {
			fmt.Println("Failed to log the import steps. Reason: ", err)
		}
	}()

	_, err = c.ImportCluster(opts, responseID)
	if err != nil {
		close(done)
		return err
	}
	wg.Wait()

	return nil
}

func getFeatureSetsInfo(featureSets map[string]string) []clustermodel.FeatureSet {
	var desiredFeatureSets []clustermodel.FeatureSet
	if len(featureSets) == 0 {
		return defaultFeatureSet
	}

	for key, val := range featureSets {
		featureSet := clustermodel.FeatureSet{
			Name:     key,
			Features: strings.Split(val, ","),
		}

		desiredFeatureSets = append(desiredFeatureSets, featureSet)
	}

	return desiredFeatureSets
}
