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
	"context"
	"fmt"
	"os"
	"path"

	"gomodules.xyz/go-sh"
	corev1 "k8s.io/api/core/v1"
	kmapi "kmodules.xyz/client-go/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	dirPerm  = 0o755
	filePerm = 0o644

	kubectlCommand  = "kubectl"
	modeActive      = "active"
	modeTerminating = "terminating"
)

var (
	resourcesMain        = []string{"datastore", "pods", "binding", "gateway"}
	resourcesGateway     = []string{"hr", "pods", "configmaps", "secrets", "gatewayconfigs", "gatewaypresets"}
	resourcesMonitoring  = []string{"grafanadashboards"}
	resourcesGatewayYaml = []string{"services"}
)

func (g *clientOrgOpts) collectNamespaces() error {
	var nsList corev1.NamespaceList
	err := g.kc.List(context.TODO(), &nsList, &client.ListOptions{})
	if err != nil {
		return err
	}

	nsMap := make(map[string]bool)
	for _, ns := range nsList.Items {
		nsMap[ns.Name] = true
	}

	constructResourceGroup := func(nsName string) resourceGroup {
		gwNs := false
		monitoringNs := false
		if _, exists := nsMap[nsName+"-gw"]; exists {
			gwNs = true
		}
		if _, exists := nsMap[nsName+"-monitoring"]; exists {
			monitoringNs = true
		}
		return resourceGroup{
			clientOrgName:       nsName,
			gwNamespace:         gwNs,
			monitoringNamespace: monitoringNs,
		}
	}

	for _, ns := range nsList.Items {
		if g.org != "" && ns.Name != g.org { // if ran for specific org, ignore all other ones.
			continue
		}
		if val, exists := ns.Labels[kmapi.ClientOrgKey]; exists {
			rg := constructResourceGroup(ns.Name)

			switch val {
			case "true":
				g.activeOrganizations = append(g.activeOrganizations, rg)
			case modeTerminating:
				g.terminatingOrganizations = append(g.terminatingOrganizations, rg)
			}
		}
	}

	return nil
}

func makeArg(resource, namespace string) []any {
	return []any{"get", resource, "-n", namespace}
}

func makeArgYaml(resource, namespace string) []any {
	return []any{"get", resource, "-n", namespace}
	// return []any{"get", resource, "-n", namespace, "--output", "yaml"} // TODO
}

func getNSHeader(ns string) []byte {
	return fmt.Appendf(nil, "\n\n===== %v =====\n", ns)
}

func getResourceHeader(res string) []byte {
	return fmt.Appendf(nil, "# %s :\n", res)
}

func (g *clientOrgOpts) collectAllResources() error {
	if g.mode == "both" || g.mode == modeActive {
		err := os.MkdirAll(path.Join(g.dir, modeActive), dirPerm)
		if err != nil {
			return err
		}

		for _, org := range g.activeOrganizations {
			err := g.collectForOneOrg(org, path.Join(g.dir, modeActive))
			if err != nil {
				return err
			}
		}
	}

	if g.mode == "both" || g.mode == modeTerminating {
		err := os.MkdirAll(path.Join(g.dir, modeTerminating), dirPerm)
		if err != nil {
			return err
		}

		for _, org := range g.terminatingOrganizations {
			err := g.collectForOneOrg(org, path.Join(g.dir, modeTerminating))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *clientOrgOpts) collectForOneOrg(org resourceGroup, groupDir string) error {
	var (
		out []byte
		err error
	)
	out = getNSHeader(org.clientOrgName)
	err = os.WriteFile(path.Join(groupDir, org.clientOrgName+".txt"), out, filePerm) // empty the file first, & write
	if err != nil {
		return err
	}

	out = []byte{}
	for _, res := range resourcesMain {
		out = append(out, getResourceHeader(res)...)
		o, err := sh.Command(kubectlCommand, makeArg(res, org.clientOrgName)...).Command("/usr/bin/tail", "-1").Output()
		if err != nil {
			return err
		}
		out = append(out, o...)
	}

	if org.monitoringNamespace {
		out = append(out, getNSHeader(org.clientOrgName+"-monitoring")...)
		for _, res := range resourcesMonitoring {
			out = append(out, getResourceHeader(res)...)
			o, err := sh.Command(kubectlCommand, makeArg(res, org.clientOrgName+"-monitoring")...).Command("/usr/bin/tail", "-1").Output()
			if err != nil {
				return err
			}
			out = append(out, o...)
		}
	}

	if org.gwNamespace {
		out = append(out, getNSHeader(org.clientOrgName+"-gw")...)
		for _, res := range resourcesGateway {
			out = append(out, getResourceHeader(res)...)
			o, err := sh.Command(kubectlCommand, makeArg(res, org.clientOrgName+"-gw")...).Command("/usr/bin/tail", "-1").Output()
			if err != nil {
				return err
			}
			out = append(out, o...)
		}

		for _, res := range resourcesGatewayYaml {
			out = append(out, getResourceHeader(res)...)
			o, err := sh.Command(kubectlCommand, makeArgYaml(res, org.clientOrgName+"-gw")...).Command("/usr/bin/tail", "-1").Output()
			if err != nil {
				return err
			}
			out = append(out, o...)
		}
	}
	return g.writeContent(groupDir, org.clientOrgName, out)
}

func (g *clientOrgOpts) writeContent(groupDir, fileName string, content []byte) error {
	f, err := os.OpenFile(path.Join(groupDir, fileName+".txt"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, filePerm)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	_, err = f.Write(content)
	if err != nil {
		return err
	}
	return nil
}
