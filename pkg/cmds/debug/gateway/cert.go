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

	acmev1 "github.com/cert-manager/cert-manager/pkg/apis/acme/v1"
	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (g *gatewayOpts) collectCerts() error {
	dirCerts := path.Join(g.dir, yamlsDir, certsDir)
	err := os.MkdirAll(dirCerts, dirPerm)
	if err != nil {
		return err
	}

	var list certv1.CertificateList
	err = g.kc.List(context.TODO(), &list, client.InNamespace(g.hr.Namespace))
	if err != nil {
		return err
	}
	for _, item := range list.Items {
		err = writeYaml(&item, dirCerts)
		if err != nil {
			return err
		}
	}
	if err := g.collectOrders(); err != nil {
		return err
	}
	return g.collectChallenges()
}

func (g *gatewayOpts) collectOrders() error {
	dirOrders := path.Join(g.dir, yamlsDir, ordersDir)
	err := os.MkdirAll(dirOrders, dirPerm)
	if err != nil {
		return err
	}

	var list acmev1.OrderList
	err = g.kc.List(context.TODO(), &list, client.InNamespace(g.hr.Namespace))
	if err != nil {
		return err
	}
	for _, item := range list.Items {
		err = writeYaml(&item, dirOrders)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *gatewayOpts) collectChallenges() error {
	dirChallenges := path.Join(g.dir, yamlsDir, challengesDir)
	err := os.MkdirAll(dirChallenges, dirPerm)
	if err != nil {
		return err
	}

	var list acmev1.ChallengeList
	err = g.kc.List(context.TODO(), &list, client.InNamespace(g.hr.Namespace))
	if err != nil {
		return err
	}
	for _, item := range list.Items {
		err = writeYaml(&item, dirChallenges)
		if err != nil {
			return err
		}
	}
	return nil
}
