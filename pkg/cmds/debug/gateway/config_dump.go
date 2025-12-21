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
	"io"
	"net/http"
	"net/url"
	"os"
	"path"

	"gomodules.xyz/pointer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kmodules.xyz/client-go/tools/portforward"
)

func (g *gatewayOpts) collectConfigDump(podMeta metav1.ObjectMeta) error {
	tunnel, err := g.forwardToPort(podMeta, "pods", pointer.IntP(19000))
	if err != nil {
		return err
	}
	defer tunnel.Close() // nolint:errcheck

	// curl http://10.42.0.82:19000/config_dump?resource%3D%26mask%3D%26name_regex%3D > out.yaml
	u := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("localhost:%d", tunnel.Local),
		Path:   "/config_dump",
	}
	q := u.Query()
	q.Set("resource", "")
	q.Set("mask", "")
	q.Set("name_regex", "")
	u.RawQuery = q.Encode()
	resp, err := http.Get(u.String())
	if err != nil {
		fmt.Printf("curl failed: %v\n", err)
		return err
	}
	defer resp.Body.Close() // nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	outputFile := path.Join(g.dir, logsDir, fmt.Sprintf("%s.config_dump.json", podMeta.Name))
	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("cannot create %s: %v", outputFile, err)
	}
	defer f.Close() // nolint:errcheck
	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("cannot write %s: %v", outputFile, err)
	}
	return nil
}

func (g *gatewayOpts) forwardToPort(podMeta metav1.ObjectMeta, resource string, port *int) (*portforward.Tunnel, error) {
	tunnel := portforward.NewTunnel(
		portforward.TunnelOptions{
			Client:    g.kubeClient.CoreV1().RESTClient(),
			Config:    g.config,
			Resource:  resource,
			Namespace: podMeta.Namespace,
			Name:      podMeta.Name,
			Remote:    *port,
		})
	if err := tunnel.ForwardPort(); err != nil {
		return nil, err
	}

	return tunnel, nil
}
