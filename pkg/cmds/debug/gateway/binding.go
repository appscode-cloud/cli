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

	catalogapi "go.bytebuilders.dev/catalog/api/catalog/v1alpha1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kmapi "kmodules.xyz/client-go/api/v1"
	ofst "kmodules.xyz/offshoot-api/api/v1"
)

type Binding struct {
	Spec   BindingSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status BindingStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

type BindingSpec struct {
	SourceRef SourceRef `json:"sourceRef,omitempty" yaml:"sourceRef,omitempty"`
}

type SourceRef struct {
	Name      string `json:"name,omitempty" yaml:"name,omitempty"`
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
}

type BindingStatus struct {
	// +optional
	Gateway *ofst.Gateway `json:"gateway,omitempty"`
}

func (g *gatewayOpts) collectBindings() error {
	var uns unstructured.UnstructuredList
	uns.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   catalogapi.GroupVersion.Group,
		Version: catalogapi.GroupVersion.Version,
		Kind:    g.getKindFromResource(g.db.resource) + "Binding",
	})

	if err := g.kc.List(context.Background(), &uns); err != nil {
		return err
	}

	dirBindings := path.Join(g.dir, yamlsDir, bindingsDir)
	if err := os.MkdirAll(dirBindings, dirPerm); err != nil {
		return err
	}

	g.gw = gwInfo{
		gateways: make([]kmapi.ObjectReference, 0),
	}
	for _, b := range uns.Items {
		var binding Binding
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(b.Object, &binding); err != nil {
			return fmt.Errorf("failed to unmarshal binding %s: %w", b.GetName(), err)
		}

		src := binding.Spec.SourceRef
		if src.Name != g.db.name || src.Namespace != g.db.namespace {
			continue
		}
		if err := writeYaml(&b, dirBindings); err != nil {
			return err
		}

		if binding.Status.Gateway == nil {
			continue
		}
		for _, ui := range binding.Status.Gateway.UI {
			if ui.HelmRelease != nil {
				g.gw.gateways = append(g.gw.gateways, kmapi.ObjectReference{
					Name:      ui.HelmRelease.Name,
					Namespace: b.GetNamespace(),
				})
				g.gw.uiReleases = append(g.gw.uiReleases, kmapi.ObjectReference{
					Name:      ui.HelmRelease.Name,
					Namespace: b.GetNamespace(),
				})
			}
		}
		g.gw.gateways = append(g.gw.gateways, kmapi.ObjectReference{
			Name:      binding.Status.Gateway.Name,
			Namespace: binding.Status.Gateway.Namespace,
		})
	}

	return nil
}
