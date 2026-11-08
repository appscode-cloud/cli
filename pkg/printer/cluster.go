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

package printer

import (
	"fmt"
	"os"
	"text/tabwriter"

	clusterv2 "go.bytebuilders.dev/b3/apis/gen/go/akp/cluster/v2"

	"google.golang.org/protobuf/encoding/protojson"
	"sigs.k8s.io/yaml"
)

var OutputFormat string

type clusterPrinter interface {
	printCluster(cluster *clusterv2.Cluster) error
	printClusterList(clusters []*clusterv2.Cluster) error
}

func newPrinter() clusterPrinter {
	var printer clusterPrinter
	switch OutputFormat {
	case "json":
		printer = &jsonPrinter{}
	case "yaml":
		printer = &yamlPrinter{}
	default:
		printer = &tablePrinter{}
	}
	return printer
}

func PrintCluster(cluster *clusterv2.Cluster) error {
	printer := newPrinter()
	return printer.printCluster(cluster)
}

func PrintClusterList(clusters []*clusterv2.Cluster) error {
	printer := newPrinter()
	return printer.printClusterList(clusters)
}

type tablePrinter struct{}

func (p *tablePrinter) printCluster(cluster *clusterv2.Cluster) error {
	return p.printClusterList([]*clusterv2.Cluster{cluster})
}

func (p *tablePrinter) printClusterList(clusters []*clusterv2.Cluster) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tDISPLAY_NAME\tPROVIDER\tPHASE")
	for _, c := range clusters {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", c.GetName(), c.GetDisplayName(), c.GetProvider(), c.GetStatus().GetPhase())
	}
	return w.Flush()
}

var jsonOptions = protojson.MarshalOptions{Indent: " "}

type jsonPrinter struct{}

func (p *jsonPrinter) printCluster(cluster *clusterv2.Cluster) error {
	data, err := jsonOptions.Marshal(cluster)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func (p *jsonPrinter) printClusterList(clusters []*clusterv2.Cluster) error {
	for _, c := range clusters {
		if err := p.printCluster(c); err != nil {
			return err
		}
	}
	return nil
}

type yamlPrinter struct{}

func (p *yamlPrinter) printCluster(cluster *clusterv2.Cluster) error {
	data, err := protojson.Marshal(cluster)
	if err != nil {
		return err
	}
	data, err = yaml.JSONToYAML(data)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func (p *yamlPrinter) printClusterList(clusters []*clusterv2.Cluster) error {
	for _, c := range clusters {
		if err := p.printCluster(c); err != nil {
			return err
		}
	}
	return nil
}
