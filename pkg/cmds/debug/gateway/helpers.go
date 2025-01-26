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
	"fmt"
	"os"
	"path"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	logsDir  = "logs"
	yamlsDir = "yamls"
	dirPerm  = 0o755
	filePerm = 0o644

	bindingsDir     = "bindings"
	classesDir      = "gwclasses"
	presetsDir      = "gwpresets"
	configsDir      = "gwconfigs"
	gatewaysDir     = "gateways"
	helmreleasesDir = "helmreleases"
	proxyDir        = "envoyproxies"
	backendTLSDir   = "backendtlspolicies"
	refGrantDir     = "referencegrants"
	routesDir       = "routes"
	secretsDir      = "secrets"
	certsDir        = "certs"
	ordersDir       = "orders"
	challengesDir   = "challenges"
	servicesDir     = "services"

	catalogManager = "catalog-manager"
	envoyGateway   = "envoy-gateway"
	envoy          = "envoy"
)

func writeYaml(obj client.Object, fullPath string) error {
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return err
	}

	us := &unstructured.Unstructured{Object: u}
	us.SetManagedFields(nil)

	b, err := yaml.Marshal(us)
	if err != nil {
		return err
	}
	return os.WriteFile(path.Join(fullPath, obj.GetName()+".yaml"), b, filePerm)
}

func (g *gatewayOpts) populateResourceMap() error {
	dc, err := discovery.NewDiscoveryClientForConfig(g.config)
	if err != nil {
		return err
	}
	g.resMap = make(map[string]string)

	if err := g.populate(dc, "kubedb.com/v1"); err != nil {
		return err
	}
	if err := g.populate(dc, "kubedb.com/v1alpha2"); err != nil {
		return err
	}
	return nil
}

func (g *gatewayOpts) populate(dc *discovery.DiscoveryClient, gv string) error {
	resources, err := dc.ServerResourcesForGroupVersion(gv)
	if err != nil {
		return err
	}
	for _, r := range resources.APIResources {
		if !strings.ContainsAny(r.Name, "/") {
			g.resMap[r.Name] = r.Kind
			g.resMap[r.SingularName] = r.Kind
			for _, s := range r.ShortNames {
				g.resMap[s] = r.Kind
			}
			g.resMap[r.Kind] = r.Kind
		}
	}
	return nil
}

func (g *gatewayOpts) getKindFromResource(res string) string {
	kind, exists := g.resMap[res]
	if !exists {
		_ = fmt.Errorf("resource %s not supported", res)
	}
	return kind
}
