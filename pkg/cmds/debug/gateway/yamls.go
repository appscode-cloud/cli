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
	"os"
	"path"
	"strings"

	catgwapi "go.bytebuilders.dev/catalog/api/gateway/v1alpha1"

	flux "github.com/fluxcd/helm-controller/api/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"kmodules.xyz/client-go/meta"
	dbapi "kubedb.dev/apimachinery/apis/kubedb/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwapia3 "sigs.k8s.io/gateway-api/apis/v1alpha3"
	gwapib1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func (g *gatewayOpts) collectHelmReleases() error {
	dirHelmReleases := path.Join(g.dir, yamlsDir, helmreleasesDir)
	err := os.MkdirAll(dirHelmReleases, dirPerm)
	if err != nil {
		return err
	}

	var hr flux.HelmRelease
	for _, r := range g.gw.uiReleases {
		err := g.kc.Get(context.TODO(), client.ObjectKey{Name: r.Name, Namespace: r.Namespace}, &hr)
		if err != nil {
			return err
		}
		err = writeYaml(&hr, dirHelmReleases)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *gatewayOpts) collectDatabase() error {
	var uns unstructured.Unstructured
	uns.SetGroupVersionKind(dbapi.SchemeGroupVersion.WithKind(g.getKindFromResource(g.db.resource)))
	err := g.kc.Get(context.Background(), types.NamespacedName{
		Namespace: g.db.namespace,
		Name:      g.db.name,
	}, &uns)
	if err != nil {
		log.Fatalf("failed to get database: %v", err)
	}

	return writeYaml(&uns, path.Join(g.dir, yamlsDir))
}

func (g *gatewayOpts) collectSecrets() error {
	dirSecrets := path.Join(g.dir, yamlsDir, secretsDir)
	err := os.MkdirAll(dirSecrets, dirPerm)
	if err != nil {
		return err
	}

	for _, secret := range g.gw.secrets {
		var sec corev1.Secret
		err = g.kc.Get(context.TODO(), types.NamespacedName{
			Namespace: secret.Namespace,
			Name:      secret.Name,
		}, &sec)
		if err != nil {
			return err
		}
		err = writeYaml(&sec, dirSecrets)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *gatewayOpts) collectService() error {
	dirServices := path.Join(g.dir, yamlsDir, servicesDir)
	err := os.MkdirAll(dirServices, dirPerm)
	if err != nil {
		return err
	}

	var services corev1.ServiceList
	err = g.kc.List(context.Background(), &services, client.InNamespace(g.hr.Namespace))
	if err != nil {
		return err
	}
	for _, svc := range services.Items {
		val, exist := svc.Spec.Selector[meta.NameLabelKey]
		if exist && val == envoy {
			err = writeYaml(&svc, dirServices)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *gatewayOpts) collectPresetsNConfigs() error {
	dirPresets := path.Join(g.dir, yamlsDir, presetsDir)
	err := os.MkdirAll(dirPresets, dirPerm)
	if err != nil {
		return err
	}

	dirConfigs := path.Join(g.dir, yamlsDir, configsDir)
	err = os.MkdirAll(dirConfigs, dirPerm)
	if err != nil {
		return err
	}

	main := types.NamespacedName{
		Namespace: "ace-gw",
		Name:      "ace",
	}

	var gwps catgwapi.GatewayPreset
	err = g.kc.Get(context.TODO(), main, &gwps)
	if err != nil {
		return err
	}
	err = writeYaml(&gwps, dirPresets)
	if err != nil {
		return err
	}

	var gwcfg catgwapi.GatewayConfig
	err = g.kc.Get(context.TODO(), main, &gwcfg)
	if err != nil {
		return err
	}
	err = writeYaml(&gwcfg, dirConfigs)
	if err != nil {
		return err
	}

	// ace gatewayPresets & configs done. Now focus on the self one.
	err = g.kc.Get(context.TODO(), types.NamespacedName{
		Namespace: g.db.namespace + "-gw",
		Name:      g.db.namespace,
	}, &gwps)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	err = writeYaml(&gwps, dirPresets)
	if err != nil {
		return err
	}

	ref := gwps.Spec.ParametersRef
	if ref == nil {
		return nil
	}
	err = g.kc.Get(context.TODO(), types.NamespacedName{
		Namespace: string(*ref.Namespace),
		Name:      ref.Name,
	}, &gwcfg)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	err = writeYaml(&gwcfg, dirConfigs)
	return err
}

func (g *gatewayOpts) collectReferenceGrants() error {
	dirRefGrant := path.Join(g.dir, yamlsDir, refGrantDir)
	err := os.MkdirAll(dirRefGrant, dirPerm)
	if err != nil {
		return err
	}

	var grants gwapib1.ReferenceGrantList
	err = g.kc.List(context.Background(), &grants)
	if err != nil {
		return err
	}
	for _, grant := range grants.Items {
		for _, from := range grant.Spec.From {
			if string(from.Namespace) == g.db.namespace {
				err = writeYaml(&grant, dirRefGrant)
				if err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}

func (g *gatewayOpts) collectBackendTLSPolicies() error {
	dirBackendTLS := path.Join(g.dir, yamlsDir, backendTLSDir)
	err := os.MkdirAll(dirBackendTLS, dirPerm)
	if err != nil {
		return err
	}

	var backends gwapia3.BackendTLSPolicyList
	err = g.kc.List(context.Background(), &backends)
	if err != nil {
		return err
	}
	for _, bc := range backends.Items {
		for _, target := range bc.Spec.TargetRefs {
			if strings.HasPrefix(string(target.Name), g.db.name) {
				err = writeYaml(&bc, dirBackendTLS)
				if err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}
