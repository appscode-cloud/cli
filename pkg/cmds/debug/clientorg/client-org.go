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

package clientorg

import (
	"log"
	"os"
	"path"

	catalogapi "go.bytebuilders.dev/catalog/api/catalog/v1alpha1"
	catgwapi "go.bytebuilders.dev/catalog/api/gateway/v1alpha1"

	acmev1 "github.com/cert-manager/cert-manager/pkg/apis/acme/v1"
	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	egv1a1 "github.com/envoyproxy/gateway/api/v1alpha1"
	flux "github.com/fluxcd/helm-controller/api/v2"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	kubedbscheme "kubedb.dev/apimachinery/client/clientset/versioned/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwapi "sigs.k8s.io/gateway-api/apis/v1"
	gwapia3 "sigs.k8s.io/gateway-api/apis/v1alpha3"
	gwapib1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	vgapi "voyagermesh.dev/gateway-api/apis/gateway/v1alpha1"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(catalogapi.AddToScheme(scheme))
	utilruntime.Must(catgwapi.AddToScheme(scheme))
	utilruntime.Must(kubedbscheme.AddToScheme(scheme))
	utilruntime.Must(gwapi.Install(scheme))
	utilruntime.Must(gwapia3.Install(scheme))
	utilruntime.Must(gwapib1.Install(scheme))
	utilruntime.Must(vgapi.AddToScheme(scheme))
	utilruntime.Must(egv1a1.AddToScheme(scheme))
	utilruntime.Must(flux.AddToScheme(scheme))
	utilruntime.Must(certv1.AddToScheme(scheme))
	utilruntime.Must(acmev1.AddToScheme(scheme))
}

func NewCmdClientOrg(f cmdutil.Factory) *cobra.Command {
	opt := newClientOrgOpts(f)
	cmd := &cobra.Command{
		Use:               "client-org",
		Short:             "ClientOrg related info",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			opt.dir = opt.org
			if opt.dir == "" {
				opt.dir = "client-org"
			}
			klog.Infof("The debug into will be generated in current directory under '%s' folder", opt.dir)
			return opt.run()
		},
	}

	cmd.Flags().StringVarP(&opt.org, "name", "m", "", "Client org name")
	return cmd
}

type clientOrgOpts struct {
	kc         client.Client
	config     *rest.Config
	kubeClient kubernetes.Interface
	org        string
	dir        string

	activeOrganizations      []resourceGroup
	terminatingOrganizations []resourceGroup
}

type resourceGroup struct {
	clientOrgName       string
	gwNamespace         bool
	monitoringNamespace bool
	mainResources       string
	gwResources         string
	monitoringResources string
}

func newClientOrgOpts(f cmdutil.Factory) *clientOrgOpts {
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

	return &clientOrgOpts{
		kc:                       kc,
		config:                   config,
		kubeClient:               cs,
		activeOrganizations:      []resourceGroup{},
		terminatingOrganizations: []resourceGroup{},
	}
}

func (g *clientOrgOpts) run() error {
	pwd, _ := os.Getwd()
	g.dir = path.Join(pwd, g.dir)
	err := os.MkdirAll(g.dir, dirPerm)
	if err != nil {
		return err
	}

	if err := g.collectNamespaces(); err != nil {
		return err
	}
	err = g.collectAllResources()
	if err != nil {
		return err
	}
	return nil
}
