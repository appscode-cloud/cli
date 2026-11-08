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

package cluster

import (
	"context"
	"fmt"
	"sync"

	clusterv2 "go.bytebuilders.dev/b3/apis/gen/go/akp/cluster/v2"
	"go.bytebuilders.dev/cli/pkg/akp"
	"go.bytebuilders.dev/cli/pkg/config"
	"go.bytebuilders.dev/cli/pkg/printer"

	"github.com/rs/xid"
	"github.com/spf13/cobra"
)

type operationStarter func(ctx context.Context, c *akp.Clients, owner string, op *clusterv2.OperationOptions) error

// runOperation starts a cluster operation and prints its steps, which AKP publishes on NATS
// under the operation's response ID, until it ends.
func runOperation(cmd *cobra.Command, f *config.Factory, steps string, start operationStarter) error {
	ctx := cmd.Context()
	c, err := f.Clients()
	if err != nil {
		return err
	}
	owner, err := c.Owner(ctx)
	if err != nil {
		return err
	}

	nc, err := c.NatsConnection(ctx, "ace-cli")
	if err != nil {
		return err
	}
	defer nc.Close() // nolint:errcheck

	responseID := xid.New().String()
	wg := sync.WaitGroup{}
	wg.Add(1)
	done := f.Canceller()
	go func() {
		err := printer.PrintNATSJobSteps(&wg, nc, responseID, done)
		if err != nil {
			fmt.Printf("Failed to log %s steps. Reason: %v\n", steps, err)
		}
	}()

	err = start(ctx, c, owner, &clusterv2.OperationOptions{ResponseId: responseID})
	if err != nil {
		close(done)
		return err
	}
	wg.Wait()

	return nil
}
