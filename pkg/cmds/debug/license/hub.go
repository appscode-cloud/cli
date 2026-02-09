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

package license

import (
	"context"
	"os"
	"path"

	"go.bytebuilders.dev/cli/pkg/cmds/utils"

	"gomodules.xyz/go-sh"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ocmapi "open-cluster-management.io/api/cluster/v1"
)

type spokeInfo struct {
	name string
	path string
}

func (g *licenseOpts) fun() error {
	getter := utils.NewKubeconfigGetter(scheme, g.config)

	var clusters ocmapi.ManagedClusterList
	err := g.kc.List(context.TODO(), &clusters)
	if err != nil {
		return err
	}

	for _, cluster := range clusters.Items {
		// cond types: HubAcceptedManagedCluster, ManagedClusterJoined, ManagedClusterConditionAvailable, ManagedClusterConditionClockSynced
		if !isConditionTrue(cluster.Status.Conditions, ocmapi.ManagedClusterConditionAvailable) {
			continue
		}
		kc, err := getter.GetSpokeClient(cluster.Name)
		if err != nil {
			return err
		}
		var ns corev1.Namespace
		err = kc.Get(context.TODO(), types.NamespacedName{Name: "org1"}, &ns)
		if err != nil {
			klog.Errorf("err getting namespace: %v", err)
		}
		klog.Infof("managed Cluster: %v , ns: %v found -------- Success!", cluster.Name, ns.Name)

		spokePath := path.Join(g.rootPath, cluster.Name)
		err = os.MkdirAll(spokePath, dirPerm)
		if err != nil {
			return err
		}

		si := spokeInfo{
			name: cluster.Name,
			path: spokePath,
		}
		if err := g.collectInfo(&si); err != nil {
			return err
		}
		if err := g.collectLogs(&si, "deploy/license-proxyserver", "kubeops"); err != nil {
			return err
		}
		if err := g.collectLogs(&si, "sts/kubedb-kubedb-provisioner", "kubedb"); err != nil {
			return err
		}

		if err := g.collectDatabaseInfo(&si); err != nil {
			return err
		}
	}
	return nil
}

func isConditionTrue(conditions []metav1.Condition, conditionType string) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType && condition.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

func (g *licenseOpts) collectDatabaseInfo(s *spokeInfo) error {
	fp := path.Join(g.rootPath, databaseFile)
	if s != nil {
		fp = path.Join(s.path, databaseFile)
	}
	out := []byte("\n\n===== Database status =====\n")
	args := []any{"get", "datastore", "-A"}
	if s != nil {
		args = append(args, "--kubeconfig", utils.DefaultPath)
	}
	o, err := sh.Command(kubectlCommand, args...).Output()
	if err != nil {
		return err
	}
	out = append(out, o...)

	out = append(out, []byte("\n\n===== Database yamls =====\n")...)
	args = []any{"get", "datastore", "-A", "-o", "yaml"}
	if s != nil {
		args = append(args, "--kubeconfig", utils.DefaultPath)
	}
	o, err = sh.Command(kubectlCommand, args...).Output()
	if err != nil {
		return err
	}
	out = append(out, o...)

	err = os.WriteFile(fp, out, filePerm)
	if err != nil {
		return err
	}
	return nil
}
