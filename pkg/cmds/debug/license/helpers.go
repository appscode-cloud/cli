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

package license

import (
	"fmt"
	"os"
	"path"
	"strings"

	"go.bytebuilders.dev/cli/pkg/cmds/utils"

	"gomodules.xyz/go-sh"
)

const (
	dirPerm  = 0o755
	filePerm = 0o644

	kubectlCommand = "kubectl"
	defaultDir     = "license-debug-info"
	defaultFile    = "info.txt"
	databaseFile   = "database.txt"
)

func (g *licenseOpts) collectInfo(s *spokeInfo) error {
	fp := path.Join(g.rootPath, defaultFile)
	if s != nil {
		fp = path.Join(s.path, defaultFile)
	}
	out := []byte("\n\n===== License status =====\n")
	args := []any{"get", "licensestatus"}
	if s != nil {
		args = append(args, "--kubeconfig", utils.DefaultPath)
	}
	o, err := sh.Command(kubectlCommand, args...).Output()
	if err != nil {
		return err
	}
	out = append(out, o...)

	err = os.WriteFile(fp, out, filePerm) // empty the file first, & write
	if err != nil {
		return err
	}
	// kube-system namespace
	out = []byte("\n\n===== kube-system namespace =====\n")
	args = []any{"get", "ns", "kube-system", "-o", "yaml"}
	if s != nil {
		args = append(args, "--kubeconfig", utils.DefaultPath)
	}
	o, err = sh.Command(kubectlCommand, args...).Output()
	if err != nil {
		return err
	}
	out = append(out, o...)

	// license-proxyserver-licenses secret
	out = append(out, []byte("\n\n===== License secret =====\n")...)
	args = []any{"get", "secrets", "license-proxyserver-licenses", "-n", "kubeops", "-o", "yaml"}
	if s != nil {
		args = append(args, "--kubeconfig", utils.DefaultPath)
	}
	o, err = sh.Command(kubectlCommand, args...).Output()
	if err != nil {
		return err
	}
	out = append(out, o...)
	err = g.writeContent(fp, out)
	if err != nil {
		return err
	}
	return nil
}

func (g *licenseOpts) collectLogs(s *spokeInfo, res, ns string) error {
	parts := strings.Split(res, "/")
	if len(parts) != 2 {
		return fmt.Errorf("unexpected res: %s", parts)
	}

	fp := path.Join(g.rootPath, parts[1]+".log")
	if s != nil {
		fp = path.Join(s.path, parts[1]+".log")
	}

	args := []any{"logs", res, "-n", ns}
	if s != nil {
		args = append(args, "--kubeconfig", utils.DefaultPath)
	}
	out, err := sh.Command(kubectlCommand, args...).Output()
	if err != nil {
		return err
	}

	err = g.writeContent(fp, out)
	if err != nil {
		return err
	}
	return nil
}

func (g *licenseOpts) writeContent(fp string, content []byte) error {
	f, err := os.OpenFile(fp, os.O_APPEND|os.O_CREATE|os.O_WRONLY, filePerm)
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
