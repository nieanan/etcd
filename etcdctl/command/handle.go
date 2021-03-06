// Copyright 2015 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package command

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/coreos/etcd/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/coreos/etcd/Godeps/_workspace/src/github.com/coreos/go-etcd/etcd"
)

type handlerFunc func(*cli.Context, *etcd.Client) (*etcd.Response, error)
type printFunc func(*etcd.Response, string)
type contextualPrintFunc func(*cli.Context, *etcd.Response, string)

// dumpCURL blindly dumps all curl output to os.Stderr
func dumpCURL(client *etcd.Client) {
	client.OpenCURL()
	for {
		fmt.Fprintf(os.Stderr, "Curl-Example: %s\n", client.RecvCURL())
	}
}

// rawhandle wraps the command function handlers and sets up the
// environment but performs no output formatting.
func rawhandle(c *cli.Context, fn handlerFunc) (*etcd.Response, error) {
	endpoints, err := getEndpoints(c)
	if err != nil {
		return nil, err
	}

	tr, err := getTransport(c)
	if err != nil {
		return nil, err
	}

	client := etcd.NewClient(endpoints)
	client.SetTransport(tr)

	if c.GlobalBool("debug") {
		go dumpCURL(client)
	}

	// Sync cluster.
	if !c.GlobalBool("no-sync") {
		if ok := client.SyncCluster(); !ok {
			handleError(ExitBadConnection, errors.New("cannot sync with the cluster using endpoints "+strings.Join(endpoints, ", ")))
		}
	}

	if c.GlobalBool("debug") {
		fmt.Fprintf(os.Stderr, "Cluster-Endpoints: %s\n", strings.Join(client.GetCluster(), ", "))
	}

	// Execute handler function.
	return fn(c, client)
}

// handlePrint wraps the command function handlers to parse global flags
// into a client and to properly format the response objects.
func handlePrint(c *cli.Context, fn handlerFunc, pFn printFunc) {
	resp, err := rawhandle(c, fn)

	// Print error and exit, if necessary.
	if err != nil {
		handleError(ExitServerError, err)
	}

	if resp != nil && pFn != nil {
		pFn(resp, c.GlobalString("output"))
	}
}

// Just like handlePrint but also passed the context of the command
func handleContextualPrint(c *cli.Context, fn handlerFunc, pFn contextualPrintFunc) {
	resp, err := rawhandle(c, fn)

	if err != nil {
		handleError(ExitServerError, err)
	}

	if resp != nil && pFn != nil {
		pFn(c, resp, c.GlobalString("output"))
	}
}

// handleDir handles a request that wants to do operations on a single dir.
// Dir cannot be printed out, so we set NIL print function here.
func handleDir(c *cli.Context, fn handlerFunc) {
	handlePrint(c, fn, nil)
}

// handleKey handles a request that wants to do operations on a single key.
func handleKey(c *cli.Context, fn handlerFunc) {
	handlePrint(c, fn, printKey)
}

func handleAll(c *cli.Context, fn handlerFunc) {
	handlePrint(c, fn, printAll)
}
