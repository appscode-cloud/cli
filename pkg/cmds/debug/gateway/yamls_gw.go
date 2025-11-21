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
	"os"
	"path"

	egv1a1 "github.com/envoyproxy/gateway/api/v1alpha1"
	flux "github.com/fluxcd/helm-controller/api/v2"
	"gomodules.xyz/pointer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	kmapi "kmodules.xyz/client-go/api/v1"
	gwapi "sigs.k8s.io/gateway-api/apis/v1"
)

func (g *gatewayOpts) collectGateways() error {
	dirGateways := path.Join(g.dir, yamlsDir, gatewaysDir)
	err := os.MkdirAll(dirGateways, dirPerm)
	if err != nil {
		return err
	}

	var gwList []gwapi.Gateway
	var gwObj gwapi.Gateway
	for _, gg := range g.gw.gateways {
		err := g.kc.Get(context.TODO(), types.NamespacedName{
			Namespace: gg.Namespace,
			Name:      gg.Name,
		}, &gwObj)
		if err != nil {
			return err
		}
		gwList = append(gwList, gwObj)
		err = writeYaml(&gwObj, dirGateways)
		if err != nil {
			return err
		}
	}

	err = g.fetchGwInfo(gwList)
	if err != nil {
		return err
	}
	// klog.Infof("gateways:%v ; classes=%v ; routes=%v ; secrets=%v \n", g.gw.gateways, g.gw.gwClasses, g.gw.routes, g.gw.secrets)
	return g.collectGWClass()
}

func (g *gatewayOpts) fetchGwInfo(gwList []gwapi.Gateway) error {
	classes := make(map[string]bool)
	routes := make(map[gwapi.RouteGroupKind]bool)
	secrets := make(map[kmapi.ObjectReference]bool)
	// fetch gatewayClass, routes & secrets from gw yamls.
	for _, gw := range gwList {
		classes[string(gw.Spec.GatewayClassName)] = true
		for _, lis := range gw.Spec.Listeners {
			for _, r := range lis.AllowedRoutes.Kinds {
				routes[r] = true
			}
			if lis.AllowedRoutes.Kinds == nil { // Default HTTPRoute
				routes[gwapi.RouteGroupKind{
					Group: (*gwapi.Group)(pointer.StringP("gateway.networking.k8s.io")),
					Kind:  "HTTPRoute",
				}] = true
			}
			if lis.TLS != nil {
				for _, certRef := range lis.TLS.CertificateRefs {
					secrets[kmapi.ObjectReference{
						Namespace: string(*certRef.Namespace),
						Name:      string(certRef.Name),
					}] = true
				}
			}
		}
	}
	i := 0
	g.gw.gwClasses = make([]string, len(classes))
	for c := range classes {
		g.gw.gwClasses[i] = c
		i++
	}
	i = 0
	g.gw.routes = make([]gwapi.RouteGroupKind, len(routes))
	for r := range routes {
		g.gw.routes[i] = r
		i++
	}
	i = 0
	g.gw.secrets = make([]kmapi.ObjectReference, len(secrets))
	for s := range secrets {
		g.gw.secrets[i] = s
		i++
	}
	return nil
}

func (g *gatewayOpts) collectGWClass() error {
	dirClasses := path.Join(g.dir, yamlsDir, classesDir)
	err := os.MkdirAll(dirClasses, dirPerm)
	if err != nil {
		return err
	}

	for _, class := range g.gw.gwClasses {
		var cls gwapi.GatewayClass
		err = g.kc.Get(context.TODO(), types.NamespacedName{
			Name: class,
		}, &cls)
		if err != nil {
			return err
		}
		err = writeYaml(&cls, dirClasses)
		if err != nil {
			return err
		}

		err = g.collectHelmRelease(cls)
		if err != nil {
			return err
		}
		err = g.collectEnvoyProxy(cls)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *gatewayOpts) collectHelmRelease(cls gwapi.GatewayClass) error {
	hrName := cls.Annotations["meta.helm.sh/release-name"]
	hrNamespace := cls.Annotations["meta.helm.sh/release-namespace"]
	klog.Infof("Found gatewayClass: %v %v", cls.Name, cls.Annotations)
	if hrName != "" && hrNamespace != "" {
		g.hr = types.NamespacedName{
			Namespace: hrNamespace,
			Name:      hrName,
		}
		var hr flux.HelmRelease
		err := g.kc.Get(context.TODO(), g.hr, &hr)
		if err != nil {
			return err
		}
		err = writeYaml(&hr, path.Join(g.dir, yamlsDir, helmreleasesDir))
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *gatewayOpts) collectEnvoyProxy(cls gwapi.GatewayClass) error {
	ref := cls.Spec.ParametersRef
	if ref == nil {
		return nil
	}
	dirProxy := path.Join(g.dir, yamlsDir, proxyDir)
	err := os.MkdirAll(dirProxy, dirPerm)
	if err != nil {
		return err
	}

	var ep egv1a1.EnvoyProxy
	err = g.kc.Get(context.TODO(), types.NamespacedName{
		Namespace: string(*ref.Namespace),
		Name:      ref.Name,
	}, &ep)
	if err != nil {
		return err
	}
	err = writeYaml(&ep, dirProxy)
	return err
}

func (g *gatewayOpts) collectSeedBackendInfo() error {
	klog.Infof("%v, %v", g.hr.Namespace, g.hr.Name)
	seedKey := types.NamespacedName{
		Namespace: g.hr.Namespace,
		Name:      g.hr.Namespace + "-seed-backend",
	}
	dirGateways := path.Join(g.dir, yamlsDir, gatewaysDir)
	var gwObj gwapi.Gateway
	err := g.kc.Get(context.TODO(), seedKey, &gwObj)
	if err != nil {
		return err
	}
	err = writeYaml(&gwObj, dirGateways)
	if err != nil {
		return err
	}

	dirRoutes := path.Join(g.dir, yamlsDir, routesDir)
	var ht gwapi.HTTPRoute
	err = g.kc.Get(context.TODO(), seedKey, &ht)
	if err != nil {
		return err
	}
	err = writeYaml(&ht, dirRoutes)
	return err
}
