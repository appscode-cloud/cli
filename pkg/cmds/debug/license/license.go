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
	"log"
	"os"
	"path"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	kubedbscheme "kubedb.dev/apimachinery/client/clientset/versioned/scheme"
	ocmapi "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(kubedbscheme.AddToScheme(scheme))
	utilruntime.Must(ocmapi.Install(scheme))
}

func NewCmdLicense(f cmdutil.Factory) *cobra.Command {
	opt := newLicenseOrgOpts(f)
	cmd := &cobra.Command{
		Use:               "license",
		Short:             "License related info",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			klog.Infof("The debug info will be generated in current directory under '%s' folder", defaultDir)
			_ = opt.run()
			return opt.fun()
		},
	}
	return cmd
}

type licenseOpts struct {
	kc         client.Client
	config     *rest.Config
	kubeClient kubernetes.Interface
	rootPath   string
}

func newLicenseOrgOpts(f cmdutil.Factory) *licenseOpts {
	config, err := f.ToRESTConfig()
	if err != nil {
		log.Fatal(err)
	}
	kc, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("failed to create kube client: %v", err)
	}

	return &licenseOpts{
		kc:         kc,
		config:     config,
		kubeClient: cs,
	}
}

func (g *licenseOpts) run() error {
	pwd, _ := os.Getwd()
	g.rootPath = path.Join(pwd, defaultDir)
	err := os.MkdirAll(g.rootPath, dirPerm)
	if err != nil {
		return err
	}

	if err := g.collectInfo(nil); err != nil {
		return err
	}
	if err := g.collectLogs(nil, "deploy/license-proxyserver", "kubeops"); err != nil {
		return err
	}
	if err := g.collectLogs(nil, "sts/kubedb-kubedb-provisioner", "kubedb"); err != nil {
		return err
	}

	if err := g.collectLogs(nil, "deploy/license-proxyserver-manager", "open-cluster-management-addon"); err != nil {
		return err
	}
	return err
}
