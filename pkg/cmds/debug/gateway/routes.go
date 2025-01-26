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
	"fmt"
	"os"
	"path"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ParentRef struct {
	Name      string `json:"name,omitempty" yaml:"name,omitempty"`
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
}

type RouteSpec struct {
	ParentRefs []ParentRef `json:"parentRefs,omitempty" yaml:"parentRefs,omitempty"`
}

type Route struct {
	Spec RouteSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
}

func (g *gatewayOpts) collectRoutes() error {
	dirRoutes := path.Join(g.dir, yamlsDir, routesDir)
	if err := os.MkdirAll(dirRoutes, dirPerm); err != nil {
		return err
	}

	for _, route := range g.gw.routes {
		version := "v1"
		if string(*route.Group) == "gateway.voyagermesh.com" { // our custom group
			version = "v1alpha1"
		}

		var uns unstructured.UnstructuredList
		uns.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   string(*route.Group),
			Version: version,
			Kind:    string(route.Kind),
		})
		if err := g.kc.List(context.Background(), &uns); err != nil {
			return err
		}
		for _, un := range uns.Items {
			var rt Route
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, &rt); err != nil {
				return fmt.Errorf("unmarshal route %s/%s: %w",
					un.GetNamespace(), un.GetName(), err)
			}

			// ---- b) Filter by parentRefs ----
			keep := false
			for _, pr := range rt.Spec.ParentRefs {
				if g.matchesGateway(pr) {
					keep = true
					break
				}
			}
			if !keep {
				continue // not attached to our gateway
			}

			// ---- c) Write the *original* Unstructured (preserves all fields) ----
			if err := writeYaml(&un, dirRoutes); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *gatewayOpts) matchesGateway(ref ParentRef) bool {
	for _, gateway := range g.gw.gateways {
		if ref.Name == gateway.Name && ref.Namespace == gateway.Namespace {
			return true
		}
	}
	return false
}
