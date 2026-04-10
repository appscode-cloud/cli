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

package gateway

import (
	"context"
	"log"

	catalogapi "go.bytebuilders.dev/catalog/api/catalog/v1alpha1"
	catgwapi "go.bytebuilders.dev/catalog/api/gateway/v1alpha1"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	kutil "kmodules.xyz/client-go"
	cu "kmodules.xyz/client-go/client"
	dbapi "kubedb.dev/apimachinery/apis/kubedb/v1"
	kubedbscheme "kubedb.dev/apimachinery/client/clientset/versioned/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(catalogapi.AddToScheme(scheme))
	utilruntime.Must(catgwapi.AddToScheme(scheme))
	utilruntime.Must(kubedbscheme.AddToScheme(scheme))
}

func NewCmdGateway(f cmdutil.Factory) *cobra.Command {
	opt := newGatewayOpts(f)
	cmd := &cobra.Command{
		Use:               "gateway",
		Short:             "Gateway related info",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return opt.run()
		},
	}

	cmd.Flags().StringVarP(&opt.db.resource, "db-type", "t", "mongodb", "Database type")
	cmd.Flags().StringVarP(&opt.db.name, "name", "m", "mg-test", "Database name")
	cmd.Flags().StringVarP(&opt.db.namespace, "namespace", "n", "demo", "Database namespace")
	return cmd
}

type gatewayOpts struct {
	kc     client.Client
	disc   *discovery.DiscoveryClient
	config *rest.Config
	db     dbInfo

	mapResourceToKind map[string]string
	mapSingularToKind map[string]string
}

type dbInfo struct {
	resource  string
	name      string
	namespace string
}

func newGatewayOpts(f cmdutil.Factory) *gatewayOpts {
	config, err := f.ToRESTConfig()
	if err != nil {
		log.Fatal(err)
	}
	kc, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	disc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		log.Fatal("creating discovery client: %w", err)
	}

	return &gatewayOpts{kc: kc, config: config, disc: disc}
}

func (g *gatewayOpts) run() error {
	err := g.initMap()
	if err != nil {
		return err
	}
	kind := g.resolveKind(g.db.resource) + "Binding"
	var binding unstructured.Unstructured
	binding.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   catalogapi.GroupVersion.Group,
		Version: catalogapi.GroupVersion.Version,
		Kind:    kind,
	})
	binding.SetNamespace(g.db.namespace)
	binding.SetName(g.db.name)

	// Set spec.sourceRef
	sourceRef := map[string]any{
		"name": g.db.name,
	}

	if g.db.namespace != "" {
		sourceRef["namespace"] = g.db.namespace
	}

	if err := unstructured.SetNestedField(binding.Object, sourceRef, "spec", "sourceRef"); err != nil {
		return err
	}

	vt, err := cu.CreateOrPatch(context.TODO(), g.kc, &binding, func(obj client.Object, createOp bool) client.Object {
		return obj
		// in := obj.(*catalogapi.BindingInterface)
		// return in
	})
	if vt != kutil.VerbUnchanged {
		klog.Infof("%s/%s of kind %s has been %s", g.db.namespace, g.db.name, kind, vt)
	}
	return err
}

func (g *gatewayOpts) initMap() error {
	preferredResources, err := g.disc.ServerPreferredResources()
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return err
	}

	g.mapSingularToKind = make(map[string]string)
	g.mapResourceToKind = make(map[string]string)

	for _, p := range preferredResources {
		// if p.GroupVersionKind().Group != dbapi.SchemeGroupVersion.Group {continue }
		// This can't be done. Cause p.GroupVersionKind() is empty somehow
		for _, res := range p.APIResources {
			if res.Group != dbapi.SchemeGroupVersion.Group {
				continue
			}
			g.entry(res)
		}
	}
	return nil
}

func (g *gatewayOpts) entry(res metav1.APIResource) {
	g.mapResourceToKind[res.Name] = res.Kind
	g.mapSingularToKind[res.SingularName] = res.Kind
}

func (g *gatewayOpts) resolveKind(s string) string {
	val, exists := g.mapSingularToKind[s]
	if exists {
		return val
	}
	val, exists = g.mapResourceToKind[s]
	if exists {
		return val
	}
	return s
}
